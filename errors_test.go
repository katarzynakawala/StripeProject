package stripe_test

import (
	"encoding/json"
	"testing"

	stripe "github.com/katarzynakawala/StripeProject"
)

var errorJSON = []byte(
	`{
		"error": {
		  "code": "resource_missing",
		  "doc_url": "https://stripe.com/docs/error-codes/resource-missing",
		  "message": "No such customer: 'cus_123'",
		  "param": "customer",
		  "type": "invalid_request_error"
		}
	}`)

//func TestError_Marshal(t *testing.T) {

//}

func TestError_Unmarshal(t *testing.T) {
	var se stripe.Error
	err := json.Unmarshal(errorJSON, &se)
	if err != nil {
		t.Fatalf("Unmarshal() err =%v; want nil", err)
	}

	wantDocUrl :=  "https://stripe.com/docs/error-codes/resource-missing"
	if se.DocURL != wantDocUrl {
		t.Errorf("DocURL = %s; want %s", se.DocURL, wantDocUrl)
	}

	wantType :=  "invalid_request_error"
	if se.Type != wantType {
		t.Errorf("Type = %s; want %s", se.Type, wantType)
	}

	wantMessage :=  "No such customer: 'cus_123'"
	if se.Message != wantMessage {
		t.Errorf("Message = %s; want %s", se.Message, wantMessage)
	}
}
