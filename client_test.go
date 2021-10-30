package stripe_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/katarzynakawala/StripeProject"
)

var (
	apiKey string
	//"sk_test_4eC39HqLyjWDarjtT1zdp7dc"
)

func init() {
	flag.StringVar(&apiKey, "key", "", "Your Test secret key for the Stipre Api. If present, integration tests will be run using this key.")
}
 func TestClient_Customer(t *testing.T) {
	if apiKey == "" {
		t.Skip("No API key provided")
	}


	c := stripe.Client{
		Key: apiKey,
	}
 	tok := "tok_amex"

 	cus, err := c.Customer(tok)
 	if err != nil {
 		t.Errorf("Customer() err = %v; want %v", err, nil)
 	}

 	if cus == nil {
 		t.Fatalf("Customer() = nil; wanted non-nil value")
 	}

 	if !strings.HasPrefix(cus.ID, "cus_") {
 		t.Errorf("Customer() ID = %s; want prefix %q", cus.ID, "cus_")
 	}
 }