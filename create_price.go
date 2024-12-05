package main

import (
  "fmt"
	"os"
	"github.com/joho/godotenv"
  "github.com/stripe/stripe-go/v81"
  "github.com/stripe/stripe-go/v81/product"
  "github.com/stripe/stripe-go/v81/price"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		fmt.Println("Error while loading .env:", err)
	}

  stripe.Key = os.Getenv("STRIPE_KEY")

	product_params := &stripe.ProductParams{
		Name:        stripe.String("Starter Subscription"),
		Description: stripe.String("$12/Month subscription"),
	}

	starter_product, _ := product.New(product_params)

	price_params := &stripe.PriceParams{
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Product:  stripe.String(starter_product.ID),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
		},
		UnitAmount: stripe.Int64(1200),
	}

	starter_price, _ := price.New(price_params)

	fmt.Println("Success! Here is your starter subscription product id: " + starter_product.ID)
	fmt.Println("Success! Here is your starter subscription price id: " + starter_price.ID)
}