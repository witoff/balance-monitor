package main

import (
	"bufio"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	net_url "net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

)

// One Crypto Key
type Key struct {
	Address       string
	Name		  string
	value         float64
	success       bool
	error_message string
	check_time    int64
}

// Block Explorer
type Explorer struct {
	Name        string
	Url_pattern string
	Regexp      string 
}

// Email Alert
type EmailAlert struct {
	From string
	To   string 
}

// Configuration 
type Config struct {
	Explorer     Explorer
	Min_value    float64
	Ltc_keys     []Key
	Email_alert  EmailAlert
}

// Command Line Options
type Options struct {
	verbose     bool
	debug       bool
	config_file string
}

var options Options
var config Config

// Parse Command Line Options
func ParseArgs() {
	options = Options{}

	// Define Flags
	configFile := "./config.yaml"
	cPtr := flag.String("c", configFile, "[optional] set a custom config file")
	dPtr := flag.Bool("d", false, "run in debug mode")
	debugPtr := flag.Bool("debug", false, "")
	vPtr := flag.Bool("v", false, "whether to run verbosely")
	verbosePtr := flag.Bool("verbose", false, "")

	// Parse Flags
	flag.Parse()
	options.verbose = *verbosePtr || *vPtr
	options.debug = *debugPtr || *dPtr
	options.config_file = *cPtr
}

// Parse Config File
func ParseConfig() Config {
	// Read File
	configData, err := ioutil.ReadFile(options.config_file)
	check(err)

	// Parse YAML
	parsedConfig := Config{}
	err = yaml.Unmarshal([]byte(configData), &parsedConfig)
	check(err)
	return parsedConfig
}

// One-liner to panic on error
func check(e error) {
	if e != nil {
		log.Fatalf("error: %v", e)
		panic(e)
	}
}

func main() {
	ParseArgs()
	config = ParseConfig()

	keys := make([]*Key, len(config.Ltc_keys))

	// Concurrently look up value
	concurrency := 20
	if options.debug {
		concurrency = 1
	}

	sem := make(chan bool, concurrency)
	for i, key := range config.Ltc_keys {
		sem <- true
		go func(i int, key Key) {
			defer func() { <-sem }()
			keys[i] = ProcessKey(config.Explorer, key)

		}(i, key)

	}
	// wait for goroutines to finish
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	// Alert on values below minimum
	for _, k := range keys { 
		if k.value < config.Min_value  {
			fmt.Println("- Alert!", k.Name)
			SendAlert(config.Email_alert, *k)
		}
	}

	// Print verbose data
	for _, k := range keys {
		if !k.success || options.verbose {
			fmt.Println("request status via: ", config.Explorer.Name)
			fmt.Printf("- key:           %v\n", k.Address)
			fmt.Printf("- name:          %v\n", k.Name)
			fmt.Printf("- success:       %v\n", k.success)
			fmt.Printf("- value:         %v\n", k.value)
			fmt.Printf("- error message: %v\n", k.error_message)
		}
	}
	fmt.Println("Done")
}

// Lookup value for a single key
func ProcessKey(explorer Explorer, key Key) *Key {
	if options.verbose {
		fmt.Println("Processing Key:", key.Name)
	}

	key.success = false

	// Get url data
	request_url := fmt.Sprintf(explorer.Url_pattern, key.Address)
	resp, err := http.Get(request_url)

	// Errors don't indicate a version change.  Fail on them.
	if err != nil || resp.StatusCode != 200 {
		statuscode := 0
		if resp != nil {
			statuscode = resp.StatusCode
		}
		key.error_message = fmt.Sprintf("msg: %v, status code: %v", err, statuscode)
		return &key
	}

	data, err := ioutil.ReadAll(resp.Body)
	body := string(data)
	resp.Body.Close()

	key.value, err = ExtractValue(body, explorer)
	check(err)

	// Verbose & Debug Info
	if options.verbose {
		fmt.Println("- key: ", key.Address)
	}

	if options.debug {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("<Press Enter To Continue>")
		reader.ReadString('\n')
	}

	key.success = true 
	key.check_time = time.Now().Unix()
	return &key
}

func ExtractValue(body string, explorer Explorer) (float64, error) {
	re := regexp.MustCompile(explorer.Regexp)
	matches := re.FindAllStringSubmatch(body, -1)

	if len(matches) == 0 || len(matches) > 1 {
		fmt.Printf("Warning: %v regex captures found\n", len(matches))
		return -1, nil
	}
	return strconv.ParseFloat(matches[0][1], 64)
}

func GetHost(url string) string {
	host, err := net_url.Parse(url)
	check(err)
	return host.Host
}

func SendAlert(email_alert EmailAlert, key Key) {

	from := mail.NewEmail("Balance Monitor", email_alert.From)
	subject := fmt.Sprintf("Zero Balance Key %v", key.Name) 
	to := mail.NewEmail(email_alert.To, email_alert.To)
	plainTextContent := fmt.Sprintf("%v key %v balance is %v", key.Name, key.Address, key.value)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, plainTextContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	check(err)

	fmt.Println(response.StatusCode)
	fmt.Println(response.Body)
	fmt.Println(response.Headers)
}