package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"os"
	"github.com/joho/godotenv"
  "github.com/stripe/stripe-go/v81"
  "github.com/stripe/stripe-go/v81/customer"
  "github.com/stripe/stripe-go/v81/subscription"
	_ "github.com/lib/pq"
)

// Definição das estruturas de dados
type User struct {
	ID        int
	Name      string
	Email     string
	StripeID  string
}

type Subscription struct {
	ID             int
	UserID         int
	StripeSubID    string
	PlanID         string
	Status         string
	StartDate      time.Time
	CancelDate     sql.NullTime
	NextBillingDay time.Time
}

type Payment struct {
	ID             int
	SubscriptionID int
	Amount         float64
	Status         string
	PaymentDate    time.Time
	InvoiceURL     string
}

// Função para configurar o banco de dados PostgreSQL
func setupDatabase() *sql.DB {
	connStr := "user=youruser dbname=yourdb sslmode=disable password=yourpassword"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Criação das tabelas
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100) UNIQUE,
			stripe_id VARCHAR(50)
		);

		CREATE TABLE IF NOT EXISTS subscriptions (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			stripe_sub_id VARCHAR(50),
			plan_id VARCHAR(50),
			status VARCHAR(20),
			start_date TIMESTAMP,
			cancel_date TIMESTAMP,
			next_billing_day TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS payments (
			id SERIAL PRIMARY KEY,
			subscription_id INTEGER REFERENCES subscriptions(id),
			amount DECIMAL(10,2),
			status VARCHAR(20),
			payment_date TIMESTAMP,
			invoice_url VARCHAR(255)
		);
	`)

	if err != nil {
		log.Fatal(err)
	}

	return db
}

// Função para criar um usuário
func createUser(db *sql.DB, name, email string) (*User, error) {
	// Cria cliente no Stripe
	params := &stripe.CustomerParams{
		Name:  stripe.String(name),
		Email: stripe.String(email),
	}
	stripeCustomer, err := customer.New(params)
	if err != nil {
		return nil, err
	}

	// Salva usuário no banco de dados
	var userID int
	err = db.QueryRow(`
		INSERT INTO users (name, email, stripe_id)
		VALUES ($1, $2, $3) RETURNING id
	`, name, email, stripeCustomer.ID).Scan(&userID)

	if err != nil {
		return nil, err
	}

	return &User{
		ID:       userID,
		Name:     name,
		Email:    email,
		StripeID: stripeCustomer.ID,
	}, nil
}

// Função para criar assinatura
func createSubscription(db *sql.DB, user *User) (*Subscription, error) {
	// Parâmetros do plano no Stripe
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(user.StripeID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String("price_100BRL_monthly"), // ID do preço no Stripe
			},
		},
	}

	// Cria assinatura no Stripe
	stripeSubscription, err := subscription.New(subscriptionParams)
	if err != nil {
		return nil, err
	}

	// Salva assinatura no banco de dados
	var subID int
	err = db.QueryRow(`
		INSERT INTO subscriptions (
			user_id, stripe_sub_id, plan_id, status,
			start_date, next_billing_day
		) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, user.ID, stripeSubscription.ID, stripeSubscription.Items.Data[0].Price.ID,
		stripeSubscription.Status,
		time.Now(),
		time.Unix(stripeSubscription.CurrentPeriodEnd, 0)).Scan(&subID)

	if err != nil {
		return nil, err
	}

	return &Subscription{
		ID:             subID,
		UserID:         user.ID,
		StripeSubID:    stripeSubscription.ID,
		PlanID:         stripeSubscription.Items.Data[0].Price.ID,
		Status:         string(stripeSubscription.Status),
		StartDate:      time.Now(),
		NextBillingDay: time.Unix(stripeSubscription.CurrentPeriodEnd, 0),
	}, nil
}

// Função para cancelar assinatura
func cancelSubscription(db *sql.DB, subscriptionID int) error {
	// Obtém a assinatura do banco de dados
	var sub Subscription
	err := db.QueryRow(`
		SELECT stripe_sub_id FROM subscriptions
		WHERE id = $1
	`, subscriptionID).Scan(&sub.StripeSubID)

	if err != nil {
		return err
	}

	// Cancela no Stripe
	_, err = subscription.Cancel(sub.StripeSubID, nil)
	if err != nil {
		return err
	}

	// Atualiza status no banco de dados
	_, err = db.Exec(`
		UPDATE subscriptions
		SET status = 'canceled',
		cancel_date = $1
		WHERE id = $2
	`, time.Now(), subscriptionID)

	return err
}

func main() {
	// Configura chave da Stripe (deve ser configurada antes)
	stripe.Key = os.Getenv("STRIPE_KEY")

	// Configura banco de dados
	db := setupDatabase()
	defer db.Close()

	// Exemplo de fluxo completo
	user, err := createUser(db, "João Silva", "joao@exemplo.com")
	if err != nil {
		log.Fatal(err)
	}

	subscription, err := createSubscription(db, user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Usuário criado: %s\n", user.Name)
	fmt.Printf("Assinatura criada, próxima cobrança em: %v\n", subscription.NextBillingDay)

	// Simulação de cancelamento
	err = cancelSubscription(db, subscription.ID)
	if err != nil {
		log.Fatal(err)
	}
}