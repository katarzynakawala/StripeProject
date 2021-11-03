package stripe_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katarzynakawala/StripeProject"
)

var (
	apiKey string
	//"sk_test_4eC39HqLyjWDarjtT1zdp7dc"
	update bool
)

const (
	tokenAmex        = "tok_amex"
	tokenInvalid     = "tok_egegege"
	tokenExpiredCard = "tok_chargeDeclinedExpiredCard"
)

func init() {
	flag.StringVar(&apiKey, "key", "", "Your Test secret key for the Stipre Api. If present, integration tests will be run using this key.")
	flag.BoolVar(&update, "update", false, "Set this flag to update the responses used in local tests. This requires that the key flag is set so that we can interact with the stripe API")
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

func stripeClient(t *testing.T) (*stripe.Client, func()) {
	teardown := make([]func(), 0)
	c := stripe.Client{
		Key: apiKey,
	}
	if update {
		rc := &recorderClient{}
		c.HttpClient = rc
		teardown = append(teardown, func() {
			for i, res := range rc.responses {
				recordResponse(t , res, i)
			}
		})
	}
	return &c, func() {
		for _, fn := range teardown {
			fn()
		}
	}
}

func recordResponse(t *testing.T, resp response, count int) {
	path := filepath.Join("testdata", filepath.FromSlash(fmt.Sprintf("%s.%d.json", 
	t.Name(), count)))
	err := os.Mkdir(filepath.Dir(path), 0700)
	if err != nil {
		t.Fatalf("failed to create the response dir: %s. err = %v", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create the response file: %s. err = %v", path, err)
	}
	defer f.Close()
	jsonBytes, err := json.MarshalIndent(resp, "", " ")
	if err != nil {
		t.Fatalf("failed to marshal JSON for response file: %s. err = %v", path, err)
	}

	_, err = f.Write(jsonBytes)
	if err != nil {
		t.Fatalf("Failed to write json bytes for response file: %s. err = %v", path, err)
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
			c, teardown := stripeClient(t)
			defer teardown()
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
	c, teardown := stripeClient(t)
	defer teardown()
	tok := tokenAmex
	email := "test@test.com"

	cus, err := c.Customer(tok, email)
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
			c, teardown := stripeClient(t)
			defer teardown()
			charge, err := c.Charge(tc.customerID, tc.amount)
			for _, check := range tc.checks {
				check(t, charge, err)
			}
		})
	}
}
