package stripe_test

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/katarzynakawala/StripeProject"
)

var (
	apiKey string
	//"sk_test_4eC39HqLyjWDarjtT1zdp7dc"
)

const (
	tokenAmex        = "tok_amex"
	tokenInvalid     = "tok_egegege"
	tokenExpiredCard = "tok_chargeDeclinedExpiredCard"
)

func init() {
	flag.StringVar(&apiKey, "key", "", "Your Test secret key for the Stipre Api. If present, integration tests will be run using this key.")
}

func TestClient_Local(t *testing.T) {
	//creation of the test server 
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{
			"id": "cus_4QE0v7gcdox28x",
			"object": "customer",
			"address": null,
			"balance": 0,
			"created": 1405636867,
			"currency": "usd",
			"default_source": "card_14HNYk2eZvKYlo2C0zXVbXp3",
			"delinquent": true,
			"description": "My First Test Customer (created for API docs)",
			"discount": null,
			"email": null,
			"invoice_prefix": "3A30AC2",
			"invoice_settings": {
			  "custom_fields": null,
			  "default_payment_method": null,
			  "footer": null
			},
			"livemode": false,
			"metadata": {
			  "order_id": "6735"
			},
			"name": null,
			"next_invoice_sequence": 46690,
			"phone": null,
			"preferred_locales": [],
			"shipping": null,
			"tax_exempt": "none"
		  }`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	//test client 
	c := stripe.Client{
		Key: "wrong-key",
		BaseURL: server.URL,
	}

	_, err := c.Customer("random token", "random email")
	if err != nil {
		t.Fatalf("err = %v; want nil", err)
	}
}



func TestClient_Customer(t *testing.T) {
	if apiKey == "" {
		t.Skip("No API key provided")
	}

	type checkFn func(*testing.T, *stripe.Customer, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasNoErr := func() checkFn {
		return func(t *testing.T, cus *stripe.Customer, err error) {
			if err != nil {
				t.Fatalf("err = %v; want nil", err)
			}
		}
	}

	hasErrType := func(typee string) checkFn {
		return func(t *testing.T, cus *stripe.Customer, err error) {
			se, ok := err.(stripe.Error)
			if !ok {
				t.Fatalf("err isn't a stripe.Error")
			}
			if se.Type != typee {
				t.Errorf("err.Type = %s; want %s", se.Type, typee)
			}
		}
	}

	hasIDPrefix := func() checkFn {
		return func(t *testing.T, cus *stripe.Customer, err error) {
			if !strings.HasPrefix(cus.ID, "cus_") {
				t.Errorf("ID = %s; want prefix %q", cus.ID, "cus_")
			}
		}
	}

	hasCardDefaultSource := func() checkFn {
		return func(t *testing.T, cus *stripe.Customer, err error) {
			if !strings.HasPrefix(cus.DefaultSource, "card_") {
				t.Errorf("DefaultSource = %s; want prefix %q", cus.DefaultSource, "card_")
			}
		}
	}

	hasEmail := func(email string) checkFn {
		return func(t *testing.T, cus *stripe.Customer, err error) {
			if cus.Email != email {
				t.Errorf("Customer() Email = %s; want %s", cus.Email, email)
			}
		}
	}

	//create stripe client
	c := stripe.Client{
		Key: apiKey,
	}

	tests := map[string]struct {
		token  string
		email  string
		checks []checkFn
	}{
		"valid customer with amex": {
			token:  tokenAmex,
			email:  "test@test.com",
			checks: check(hasNoErr(), hasIDPrefix(), hasCardDefaultSource(), hasEmail("test@test.com")),
		},
		"invalid token": {
			token:  tokenInvalid,
			email:  "test@test.com",
			checks: check(hasErrType(stripe.ErrTypeInvalidRequest)),
		},
		"card expired": {
			token:  tokenExpiredCard,
			email:  "test@test.com",
			checks: check(hasErrType(stripe.ErrTypeCardError)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cus, err := c.Customer(tc.token, tc.email)
			for _, check := range tc.checks {
				check(t, cus, err)
			}
		})
	}
}

func TestClient_Charge(t *testing.T) {
	if apiKey == "" {
		t.Skip("No API key provided")
	}

	type checkFn func(*testing.T, *stripe.Charge, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasNoErr := func() checkFn {
		return func(t *testing.T, charge *stripe.Charge, err error) {
			if err != nil {
				t.Fatalf("err = %v; want nil", err)
			}
		}
	}

	hasAmount := func(amount int) checkFn {
		return func(t *testing.T, charge *stripe.Charge, err error) {
			if charge.Amount != amount {
				t.Errorf("Amount = %d; want %d", charge.Amount, amount)
			}
		}
	}

	hasErrType := func(typee string) checkFn {
		return func(t *testing.T, charge *stripe.Charge, err error) {
			se, ok := err.(stripe.Error)
			if !ok {
				t.Fatalf("err isn't a stripe.Error")
			}
			if se.Type != typee {
				t.Errorf("err.Type = %s; want %s", se.Type, typee)
			}
		}
	}

	//create stripe client
	c := stripe.Client{
		Key: apiKey,
	}

	//Create a customer for the test
	email := "test@test.com"

	cus, err := c.Customer(tokenAmex, email)
	if err != nil {
		t.Fatalf("Customer() err =%v; want nil", err)
	}

	tests := map[string]struct {
		customerID string
		amount     int
		checks     []checkFn
	}{
		"valid charge": {
			customerID: cus.ID,
			amount:     1234,
			checks:     check(hasNoErr(), hasAmount(1234)),
		},
		"invalid customer id": {
			customerID: "cus_missing",
			amount:     1234,
			checks:     check(hasErrType(stripe.ErrTypeInvalidRequest)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			charge, err := c.Charge(tc.customerID, tc.amount)
			for _, check := range tc.checks {
				check(t, charge, err)
			}
		})
	}
}
