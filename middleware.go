/*
 * moesifmiddleware-go
 */
package moesifgin

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	moesifapi "github.com/moesif/moesifapi-go"
	"github.com/moesif/moesifapi-go/models"
)

var (
	apiClient            moesifapi.API
	debug                bool
	moesifOption         map[string]interface{}
	disableTransactionId bool
	logBody              bool
	logBodyOutgoing      bool
	appConfig            = NewAppConfig()
)

func MoesifMiddleware(configurationOption map[string]interface{}) gin.HandlerFunc {
	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifOption = configurationOption
		moesifClient(moesifOption)
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		// Create a new LogGinResponseWriter to capture the response status and body for logging
		lgw := NewLogGinResponseWriter(c.Writer)
		c.Writer = lgw

		if !disableTransactionId {
			transactionId := c.Request.Header.Get("X-Moesif-Transaction-Id")
			if len(transactionId) == 0 {
				transactionId, _ = uuid()
			}
			if len(transactionId) != 0 {
				c.Request.Header.Set("X-Moesif-Transaction-Id", transactionId)
				c.Writer.Header().Add("X-Moesif-Transaction-Id", transactionId)
			}
		}

		requestTime := time.Now().UTC()
		var body1, body2 io.ReadCloser
		var err error
		if logBody {
			// buffer the entire request body into memory for logging
			if body1, body2, err = teeBody(c.Request.Body); err != nil {
				log.Printf("Error while reading request body: %v.\n", err)
			} else {
				// Body is a ReadCloser meaning that it does not implement the Seek interface
				// It must be buffered into memory to be read more than once
				// This is a replacement reader reading the buffer for the original server handler
				c.Request.Body = body1
			}
		}

		c.Next()

		// Response Time
		responseTime := time.Now().UTC()

		shouldSkip := false
		if callback, found := moesifOption["Should_Skip"]; found {
			shouldSkip = callback.(func(*gin.Context) bool)(c)
		}

		if shouldSkip {
			if debug {
				log.Printf("Skip sending the event to Moesif")
			}
		} else {
			if debug {
				log.Printf("Sending the event to Moesif")
			}
			if logBody {
				// this is a separate ReadCloser, reading the same buffer as above for logging
				c.Request.Body = body2
			}
			sendEvent(c, lgw, requestTime, responseTime)
		}
	})
}

// Initialize the client
func moesifClient(moesifOption map[string]interface{}) {
	log.Println("moesifClient update")

	var apiEndpoint string
	var batchSize int
	var eventQueueSize int
	var timerWakeupSeconds int

	// Try to fetch the api endpoint
	if endpoint, found := moesifOption["Api_Endpoint"].(string); found {
		apiEndpoint = endpoint
	}

	// Try to fetch the event queue size
	if queueSize, found := moesifOption["Event_Queue_Size"].(int); found {
		eventQueueSize = queueSize
	}

	// Try to fetch the batch size
	if batch, found := moesifOption["Batch_Size"].(int); found {
		batchSize = batch
	}

	// Try to fetch the timer wake up seconds
	if timer, found := moesifOption["Timer_Wake_Up_Seconds"].(int); found {
		timerWakeupSeconds = timer
	}

	api := moesifapi.NewAPI(moesifOption["Application_Id"].(string), &apiEndpoint, eventQueueSize, batchSize, timerWakeupSeconds)
	api.SetEventsHeaderCallback("X-Moesif-Config-ETag", appConfig.Notify)
	apiClient = api

	//  Disable debug by default
	debug = false
	// Try to fetch the debug from the option
	if isDebug, found := moesifOption["Debug"].(bool); found {
		debug = isDebug
	}

	// Disable TransactionId by default
	disableTransactionId = false
	// Try to fetch the disableTransactionId from the option
	if isEnabled, found := moesifOption["disableTransactionId"].(bool); found {
		disableTransactionId = isEnabled
	}

	// Enable logBody by default
	logBody = true
	// Try to fetch the disableTransactionId from the option
	if isEnabled, found := moesifOption["Log_Body"].(bool); found {
		logBody = isEnabled
	}

	// run goroutine to check end point for updates
	appConfig.Go()
}

// Function to generate UUID
func uuid() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)

	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// Start Capture Outgoing Request
func StartCaptureOutgoing(configurationOption map[string]interface{}) {

	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		// Set the Capture_Outoing_Requests to true to capture outgoing request
		configurationOption["Capture_Outoing_Requests"] = true
		moesifOption = configurationOption
		moesifClient(moesifOption)
	}

	if debug {
		log.Println("Start Capturing outgoing requests")
	}
	// Enable logBody by default
	logBodyOutgoing = true
	// Try to fetch the disableTransactionId from the option
	if isEnabled, found := moesifOption["Log_Body_Outgoing"].(bool); found {
		logBodyOutgoing = isEnabled
	}

	http.DefaultTransport = DefaultTransport
}

// Update User
func UpdateUser(user *models.UserModel, configurationOption map[string]interface{}) {

	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifClient(configurationOption)
	}

	// Add event to the queue
	errUpdateUser := apiClient.QueueUser(user)
	// Log the message
	if errUpdateUser != nil {
		log.Fatalf("Error while updating user: %s.\n", errUpdateUser.Error())
	} else {
		log.Println("Update User successfully added to the queue")
	}
}

// Update Users Batch
func UpdateUsersBatch(users []*models.UserModel, configurationOption map[string]interface{}) {

	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifClient(configurationOption)
	}

	// Add event to the queue
	errUpdateUserBatch := apiClient.QueueUsers(users)
	// Log the message
	if errUpdateUserBatch != nil {
		log.Fatalf("Error while updating users in batch: %s.\n", errUpdateUserBatch.Error())
	} else {
		log.Println("Updated Users successfully added to the queue")
	}
}

// Update Company
func UpdateCompany(company *models.CompanyModel, configurationOption map[string]interface{}) {

	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifClient(configurationOption)
	}

	// Add event to the queue
	errUpdateCompany := apiClient.QueueCompany(company)
	// Log the message
	if errUpdateCompany != nil {
		log.Fatalf("Error while updating company: %s.\n", errUpdateCompany.Error())
	} else {
		log.Println("Update Company successfully added to the queue")
	}
}

// Update Companies Batch
func UpdateCompaniesBatch(companies []*models.CompanyModel, configurationOption map[string]interface{}) {

	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifClient(configurationOption)
	}

	// Add event to the queue
	errUpdateCompaniesBatch := apiClient.QueueCompanies(companies)
	// Log the message
	if errUpdateCompaniesBatch != nil {
		log.Fatalf("Error while updating companies in batch: %s.\n", errUpdateCompaniesBatch.Error())
	} else {
		log.Println("Updated companies successfully added to the queue")
	}
}

// Update Subscription
func UpdateSubscription(subscription *models.SubscriptionModel, configurationOption map[string]interface{}) {

	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifClient(configurationOption)
	}

	// Add event to the queue
	errUpdateSubscription := apiClient.QueueSubscription(subscription)
	// Log the message
	if errUpdateSubscription != nil {
		log.Fatalf("Error while updating subscription: %s.\n", errUpdateSubscription.Error())
	} else {
		log.Println("Update Subscription successfully added to the queue")
	}
}

// Update Subscriptions Batch
func UpdateSubscriptionsBatch(subscriptions []*models.SubscriptionModel, configurationOption map[string]interface{}) {
	// Call the function to initialize the moesif client and moesif options
	if apiClient == nil {
		moesifClient(configurationOption)
	}

	// Add event to the queue
	errUpdateSubscriptionsBatch := apiClient.QueueSubscriptions(subscriptions)
	// Log the message
	if errUpdateSubscriptionsBatch != nil {
		log.Fatalf("Error while updating subscriptions in batch: %s.\n", errUpdateSubscriptionsBatch.Error())
	} else {
		log.Println("Updated subscriptions successfully added to the queue")
	}
}

func sendEvent(c *gin.Context, response *logGinResponseWriter, reqTime time.Time, rspTime time.Time) {
	var apiVersion *string = nil
	if isApiVersion, found := moesifOption["Api_Version"].(string); found {
		apiVersion = &isApiVersion
	}

	// Get Request Body
	var reqBody interface{}
	var reqEncoding string
	readReqBody, reqBodyErr := ioutil.ReadAll(c.Request.Body)
	if reqBodyErr != nil {
		if debug {
			log.Printf("Error while reading request body: %s.\n", reqBodyErr.Error())
		}
	}
	reqContentLength := getContentLength(c.Request.Header, readReqBody)

	// Parse the request Body if it is not empty
	reqBody = nil
	if logBody && (len(readReqBody)) > 0 {
		reqBody, reqEncoding = parseBody(readReqBody, "Request_Body_Masks")
	}

	// Get the response body
	var respBody interface{}
	var respEncoding string
	respBodyBuf := response.Body().Bytes()
	respContentLength := getContentLength(response.Header(), respBodyBuf)

	// Parse the response Body if it is not empty
	respBody = nil
	if logBody && (len(respBodyBuf)) > 0 {
		respBody, respEncoding = parseBody(respBodyBuf, "Response_Body_Masks")
	}

	// Get URL Scheme
	if c.Request.URL.Scheme == "" {
		c.Request.URL.Scheme = "http"
	}

	// Get Metadata
	var metadata map[string]interface{} = nil
	if _, found := moesifOption["Get_Metadata"]; found {
		metadata = moesifOption["Get_Metadata"].(func(*gin.Context) map[string]interface{})(c)
	}

	// Get Event top-level variables from the configuration and the request
	userId := getConfigStringValuesForIncomingEvent("Identify_User", c)
	companyId := getConfigStringValuesForIncomingEvent("Identify_Company", c)
	sessionToken := getConfigStringValuesForIncomingEvent("Get_Session_Token", c)
	direction := "Incoming"

	// Mask Headers
	requestHeader := maskHeaders(HeaderToMap(c.Request.Header), "Request_Header_Masks")
	responseHeader := maskHeaders(HeaderToMap(response.Header()), "Response_Header_Masks")

	sendMoesifAsync(c.Request, reqTime, requestHeader, apiVersion, reqBody, &reqEncoding, reqContentLength,
		rspTime, response.status, responseHeader, respBody, &respEncoding, respContentLength,
		userId, companyId, &sessionToken, metadata, &direction)
}

// teeBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
// It returns an error if the initial slurp of all bytes fails.
func teeBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
