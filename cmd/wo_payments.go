package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var woPaymentsCmd = &cobra.Command{
	Use:   "payments <work-order-id>",
	Short: "Manage payments on a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderPaymentsList,
}

var woPaymentsListCmd = &cobra.Command{
	Use:   "list <work-order-id>",
	Short: "List payments on the work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderPaymentsList,
}

var (
	woPaymentsAddAmountCents float32
	woPaymentsAddMethod      string
)

var woPaymentsAddCmd = &cobra.Command{
	Use:   "add <work-order-id>",
	Short: "Add a payment to the work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderPaymentsAdd,
}

var woPaymentsReverseArCmd = &cobra.Command{
	Use:   "reverse-ar <work-order-id>",
	Short: "Reverse accounts-receivable for the work order payments",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderPaymentsReverseAr,
}

var woPaymentsSendToArCmd = &cobra.Command{
	Use:   "send-to-ar <work-order-id>",
	Short: "Send the work order payments to accounts receivable",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderPaymentsSendToAr,
}

func init() {
	woCmd.AddCommand(woPaymentsCmd)
	woPaymentsCmd.AddCommand(woPaymentsListCmd, woPaymentsAddCmd, woPaymentsReverseArCmd, woPaymentsSendToArCmd)

	woPaymentsAddCmd.Flags().Float32Var(&woPaymentsAddAmountCents, "amount-cents", 0, "Payment amount in cents (required)")
	_ = woPaymentsAddCmd.MarkFlagRequired("amount-cents")
	woPaymentsAddCmd.Flags().StringVar(&woPaymentsAddMethod, "method", "", "Payment method (required)")
	_ = woPaymentsAddCmd.MarkFlagRequired("method")
}

func runWorkOrderPaymentsList(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "wo", "GET", func(args []string) string { return fmt.Sprintf("/work_orders/%s/payments", args[0]) },
		func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
			resp, err := client.ShowWorkOrderPayments(ctx, id)
			if err != nil {
				return nil, err
			}
			return resp.JSON200, nil
		})
}

func runWorkOrderPaymentsAdd(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/payments", workOrderID))

	req := wenmar.CreateWorkOrderPaymentRequest{
		Payment: struct {
			AmountCents float32 `json:"amount_cents"`
			Method      string  `json:"method"`
		}{
			AmountCents: woPaymentsAddAmountCents,
			Method:      woPaymentsAddMethod,
		},
	}
	resp, err := client.CreateWorkOrderPayment(context.Background(), workOrderID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs("wo", "0")}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON201), "Payment added.", nil, opts)
}

func runWorkOrderPaymentsReverseAr(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("DELETE", fmt.Sprintf("/work_orders/%d/payments/reverse_ar", workOrderID))

	resp, err := client.ReverseWorkOrderPaymentAr(context.Background(), workOrderID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "AR reversal processed.", nil, opts)
}

func runWorkOrderPaymentsSendToAr(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/payments/send_to_ar", workOrderID))

	req := wenmar.SendWorkOrderPaymentToArRequest{}
	resp, err := client.SendWorkOrderPaymentToAr(context.Background(), workOrderID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Sent to AR.", nil, opts)
}
