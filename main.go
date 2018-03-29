package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/billglover/starling"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

func main() {

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "qSXbG5pBtFhwSf9VrypqB0hXE5hhPUGmcw0u89DwhKPYKlT0NTMpZ5mBneXakBhh"},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	client := starling.NewClient(tc)

	// Look for the 'donate' savings goal
	donationGoalUID := findSavingsGoal(ctx, client, "donate")

	// If we haven't found the 'donate' goal, create one
	if donationGoalUID == "" {
		createSavingsGoal(ctx, client)
	}

	// If we still don't have a savings goal, abort
	if donationGoalUID == "" {
		fmt.Println("unable to find or create a 'donate' savings goal, giving up")
		os.Exit(1)
	}

	// Show a summary of the savings goal
	fmt.Println()
	showSavingsGoal(ctx, client, donationGoalUID)

	// Get a list of transactions for yesterday
	fmt.Println()
	amount := roundUpTxns(ctx, client)
	fmt.Println("Total Round-Up:", amount)

	// Transfer the round-up into the savings goal
	addMoney(ctx, client, donationGoalUID, amount)

	// Show a summary of the savings goal
	fmt.Println()
	showSavingsGoal(ctx, client, donationGoalUID)
}

func showSavingsGoal(ctx context.Context, client *starling.Client, uid string) {
	sg, _, err := client.GetSavingsGoal(ctx, uid)
	if err != nil {
		fmt.Println("unable to query 'donate' savings goal:", err)
		os.Exit(1)
	}

	fmt.Println("Savings Goal:", sg.Name)
	fmt.Println("------------------------")
	fmt.Println("Currency:", sg.TotalSaved.Currency)
	fmt.Println("Target:", sg.Target.MinorUnits)
	fmt.Println("Value:", sg.TotalSaved.MinorUnits)
	fmt.Println("Percent:", sg.SavedPercentage)
	fmt.Println("------------------------")
}

func createSavingsGoal(ctx context.Context, client *starling.Client) string {
	sgreq := starling.SavingsGoalRequest{
		Name:     "donate",
		Currency: "GBP",
	}

	id, _ := uuid.NewRandom()

	sgresp, _, err := client.PutSavingsGoal(ctx, id.String(), sgreq)
	if err != nil {
		fmt.Println("unable to create 'donate' savings goals:", err)
		os.Exit(1)
	}

	return sgresp.UID
}

func findSavingsGoal(ctx context.Context, client *starling.Client, name string) string {
	gs, _, err := client.GetSavingsGoals(ctx)
	if err != nil {
		fmt.Println("unable to get savings goals:", err)
		os.Exit(1)
	}

	for _, g := range gs.SavingsGoalList {
		if g.Name == name {
			return g.UID
		}
	}
	return ""
}

func addMoney(ctx context.Context, client *starling.Client, goalUID string, amount int64) {
	id, _ := uuid.NewRandom()

	tuReq := starling.TopUpRequest{
		Amount: starling.CurrencyAndAmount{
			Currency:   "GBP",
			MinorUnits: amount,
		},
	}
	_, _, err := client.AddMoney(ctx, goalUID, id.String(), tuReq)
	if err != nil {
		fmt.Println("unable to top up 'donate' savings goal:", err)
		os.Exit(1)
	}
}

func roundUpTxns(ctx context.Context, client *starling.Client) int64 {

	var roundUp int64

	dr := &starling.DateRange{
		From: time.Now().Add(time.Hour * -24),
		To:   time.Now(),
	}
	txns, _, err := client.GetTransactions(ctx, dr)
	if err != nil {
		fmt.Println("unable to get transactions:", err)
		os.Exit(1)
	}

	fmt.Println(" #: Timestamp                    Amount    RoundUp CUR Retailer")
	fmt.Println("---------------------------------------------------------------------------")
	for i, txn := range txns.Transactions {
		if txn.Direction == "OUTBOUND" && txn.Amount < 0 && txn.Source != "INTERNAL_TRANSFER" {
			minorUnits := int64(txn.Amount * -100)
			txnRoundUp := (int64(txn.Amount) * -100) - minorUnits + 100
			roundUp += txnRoundUp

			fmt.Printf("%s: %s %10d %s %s %s\n", color.CyanString("%2d", i), color.GreenString(txn.Created), minorUnits, color.RedString("%10d", txnRoundUp), txn.Currency, txn.Narrative)
		}
	}
	fmt.Println("---------------------------------------------------------------------------")

	return roundUp
}
