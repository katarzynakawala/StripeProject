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

const (
	tokenAmex        = "tok_amex"
	tokenInvalid     = "tok_egegege"
	tokenExpiredCard = "tok_chargeDeclinedExpiredCard"
)

func init() {
	flag.StringVar(&apiKey, "key", "", "Your Test secret key for the Stipre Api. If present, integration tests will be run using this key.")
}
func TestClient_Customer(t *testing.T) {
	if apiKey == "" {
		t.Skip("No API key provided")
	}

	//create stripe client
	c := stripe.Client{
		Key: apiKey,
	}

	tests := map[string]struct {
		token string
		email string
	}{
		"valid customer with amex": {
			token: tokenAmex,
			email: "test@test.com",
		},
		"invalid token": {
			token: tokenInvalid,
			email: "test@test.com",
		},
		"card expired": {
			token: tokenExpiredCard,
			email: "test@test.com",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cus, err := c.Customer(tc.token, tc.email)
			if err != nil {
				t.Errorf("Customer() err = %v; want %v", err, nil)
			}

			if cus == nil {
				t.Fatalf("Customer() = nil; want non-nil value")
			}

			if !strings.HasPrefix(cus.ID, "cus_") {
				t.Errorf("Customer() ID = %s; want prefix %q", cus.ID, "cus_")
			}

			if !strings.HasPrefix(cus.DefaultSource, "card_") {
				t.Errorf("Customer() DefaultSource = %s; want prefix %q", cus.DefaultSource, "card_")
			}

			if cus.Email != tc.email {
				t.Errorf("Customer() Email = %s; want %s", cus.Email, tc.email)
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
			checks:     check(hasErrType("invalid_request_error")),
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
