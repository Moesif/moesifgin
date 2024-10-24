package moesifgin

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/moesif/moesifapi-go/models"
)

// Queue Event to batch send to Moesif
func sendMoesifAsync(request *http.Request, reqTime time.Time, reqHeader map[string]interface{}, apiVersion *string, reqBody interface{}, reqEncoding *string, reqContentLength *int64,
	rspTime time.Time, respStatus int, respHeader map[string]interface{}, respBody interface{}, respEncoding *string, respContentLength *int64,
	userId string, companyId string, sessionToken *string, metadata map[string]interface{},
	direction *string) {

	ip := getClientIp(request)
	uri := request.URL.Scheme + "://" + request.Host + request.URL.Path
	if request.URL.RawQuery != "" {
		uri += "?" + request.URL.RawQuery
	}

	event_request := models.EventRequestModel{
		Time:             &reqTime,
		Uri:              uri,
		Verb:             request.Method,
		ApiVersion:       apiVersion,
		IpAddress:        &ip,
		Headers:          reqHeader,
		Body:             &reqBody,
		TransferEncoding: reqEncoding,
		ContentLength:    reqContentLength,
	}
	event_response := models.EventResponseModel{
		Time:             &rspTime,
		Status:           respStatus,
		IpAddress:        nil,
		Headers:          respHeader,
		Body:             respBody,
		TransferEncoding: respEncoding,
		ContentLength:    respContentLength,
	}

	// Parse sampling percentage based on user/company to decide if the event should be sent to Moesif
	// This defaults to 100% meaning that all events are logged unless specifically configured otherwise
	samplingPercentage := getSamplingPercentage(userId, companyId)
	randomPercentage := rand.Intn(100)

	if samplingPercentage > randomPercentage {
		// Weight proportionate to sampling percentage
		var eventWeight int
		if samplingPercentage == 0 {
			eventWeight = 1
		} else {
			eventWeight = 100 / samplingPercentage
		}

		event := models.EventModel{
			Request:      event_request,
			Response:     event_response,
			SessionToken: sessionToken,
			Tags:         nil,
			UserId:       &userId,
			CompanyId:    &companyId,
			Metadata:     metadata,
			Direction:    direction,
			Weight:       &eventWeight,
		}

		errSendEvent := apiClient.QueueEvent(&event)
		if errSendEvent != nil {
			log.Fatalf("Error while adding event to Moesif: %s.\n", errSendEvent.Error())
		} else {
			if debug {
				log.Println("Event successfully added to the queue")
			}
		}
	} else {
		if debug {
			log.Println("Skipped Event due to sampling percentage: " + strconv.Itoa(samplingPercentage) + " and random percentage: " + strconv.Itoa(randomPercentage))
		}
	}
}
