package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/rayli09/schwab-cli/internal/client"
)

// readOrderSpec loads an order specification (the JSON object schwab-py
// accepts as order_spec) from --file, --json, or stdin, in that order.
func readOrderSpec(file, inline string) (any, error) {
	var data []byte
	var err error
	switch {
	case file != "":
		data, err = os.ReadFile(file)
		if err != nil {
			return nil, err
		}
	case inline != "":
		data = []byte(inline)
	default:
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
	}
	var spec any
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("order spec is not valid JSON: %w", err)
	}
	return spec, nil
}

// orderSpecFlags registers the shared order-spec input flags.
func orderSpecFlags(cmd *cobra.Command, file, inline *string) {
	cmd.Flags().StringVar(file, "file", "", "read the order spec JSON from this file")
	cmd.Flags().StringVar(inline, "json", "", "order spec as an inline JSON string")
	cmd.MarkFlagsMutuallyExclusive("file", "json")
}

// dryRunGuard implements the order safety rail: without --execute the
// command prints the request it would send and returns true (handled).
func dryRunGuard(execute bool, method, path string, spec any) (bool, error) {
	if execute {
		return false, nil
	}
	out := map[string]any{
		"dry_run": true,
		"method":  method,
		"path":    path,
		"body":    spec,
		"note":    "nothing was sent; pass --execute to send this request",
	}
	enc, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return true, err
	}
	fmt.Println(string(enc))
	return true, nil
}

func registerTraderCommands(root *cobra.Command) {
	////////////////////////////////////////////////////////////////////////
	// Accounts

	var accountFields []string
	getAccountCmd := &cobra.Command{
		Use:   "get-account <account-hash>",
		Short: "Account balances/positions/orders for an account (schwab-py: get_account)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := parseEnumList(accountFields, "fields",
				func(s string) (client.AccountFields, error) {
					f := client.AccountFields(s)
					if !f.Valid() {
						return "", fmt.Errorf("invalid value %q", s)
					}
					return f, nil
				})
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetAccount(ctx, args[0], fields)
			})
		},
	}
	getAccountCmd.Flags().StringSliceVar(&accountFields, "fields", nil,
		"account fields to include (only value: positions)")
	root.AddCommand(getAccountCmd)

	root.AddCommand(&cobra.Command{
		Use:   "get-account-numbers",
		Short: "Map account numbers to account hashes (schwab-py: get_account_numbers)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetAccountNumbers(ctx)
			})
		},
	})

	var accountsFields []string
	getAccountsCmd := &cobra.Command{
		Use:   "get-accounts",
		Short: "Balances/positions/orders for all linked accounts (schwab-py: get_accounts)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := parseEnumList(accountsFields, "fields",
				func(s string) (client.AccountFields, error) {
					f := client.AccountFields(s)
					if !f.Valid() {
						return "", fmt.Errorf("invalid value %q", s)
					}
					return f, nil
				})
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetAccounts(ctx, fields)
			})
		},
	}
	getAccountsCmd.Flags().StringSliceVar(&accountsFields, "fields", nil,
		"account fields to include (only value: positions)")
	root.AddCommand(getAccountsCmd)

	////////////////////////////////////////////////////////////////////////
	// Orders

	root.AddCommand(&cobra.Command{
		Use:   "get-order <order-id> <account-hash>",
		Short: "Get one order (schwab-py: get_order)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetOrder(ctx, args[0], args[1])
			})
		},
	})

	newOrderQueryFlags := func(cmd *cobra.Command, q *orderQueryFlags) {
		cmd.Flags().IntVar(&q.maxResults, "max-results", 0,
			"maximum number of orders to return")
		cmd.Flags().StringVar(&q.from, "from-entered-time", "",
			"only orders entered at/after this time (default: 60 days ago)")
		cmd.Flags().StringVar(&q.to, "to-entered-time", "",
			"only orders entered at/before this time (default: now)")
		cmd.Flags().StringVar(&q.status, "status", "",
			"restrict to orders with this status (see schwab-py Order.Status)")
	}

	var oq1 orderQueryFlags
	getOrdersForAccountCmd := &cobra.Command{
		Use:   "get-orders-for-account <account-hash>",
		Short: "Orders for one account (schwab-py: get_orders_for_account)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := oq1.build()
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetOrdersForAccount(ctx, args[0], q)
			})
		},
	}
	newOrderQueryFlags(getOrdersForAccountCmd, &oq1)
	root.AddCommand(getOrdersForAccountCmd)

	var oq2 orderQueryFlags
	getOrdersAllCmd := &cobra.Command{
		Use:   "get-orders-for-all-linked-accounts",
		Short: "Orders for all linked accounts (schwab-py: get_orders_for_all_linked_accounts)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := oq2.build()
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetOrdersForAllLinkedAccounts(ctx, q)
			})
		},
	}
	newOrderQueryFlags(getOrdersAllCmd, &oq2)
	root.AddCommand(getOrdersAllCmd)

	var placeFile, placeJSON string
	var placeExecute bool
	placeOrderCmd := &cobra.Command{
		Use:   "place-order <account-hash>",
		Short: "Place an order — DRY-RUN unless --execute (schwab-py: place_order)",
		Long: `Place an order for an account. The order specification is the JSON
object schwab-py accepts as order_spec (see schwab-py's order builders for
its shape); supply it with --file, --json, or on stdin.

SAFETY: without --execute this command only prints the request and sends
nothing.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := readOrderSpec(placeFile, placeJSON)
			if err != nil {
				return err
			}
			path := "/trader/v1/accounts/" + args[0] + "/orders"
			if handled, err := dryRunGuard(placeExecute, "POST", path, spec); handled {
				return err
			}
			if err := requireCredentials(); err != nil {
				return err
			}
			c := newRESTClient()
			resp, err := c.PlaceOrder(context.Background(), args[0], spec)
			if err != nil {
				return err
			}
			if err := printResponse(resp); err != nil {
				return err
			}
			// Mirror schwab.utils.Utils.extract_order_id.
			if id, ok, err := client.ExtractOrderID(resp, args[0]); err != nil {
				return err
			} else if ok {
				fmt.Fprintf(os.Stderr, "order_id: %d\n", id)
			}
			return nil
		},
	}
	orderSpecFlags(placeOrderCmd, &placeFile, &placeJSON)
	placeOrderCmd.Flags().BoolVar(&placeExecute, "execute", false,
		"actually send the order (default is dry-run)")
	root.AddCommand(placeOrderCmd)

	var replaceFile, replaceJSON string
	var replaceExecute bool
	replaceOrderCmd := &cobra.Command{
		Use:   "replace-order <account-hash> <order-id>",
		Short: "Replace an order — DRY-RUN unless --execute (schwab-py: replace_order)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := readOrderSpec(replaceFile, replaceJSON)
			if err != nil {
				return err
			}
			path := "/trader/v1/accounts/" + args[0] + "/orders/" + args[1]
			if handled, err := dryRunGuard(replaceExecute, "PUT", path, spec); handled {
				return err
			}
			if err := requireCredentials(); err != nil {
				return err
			}
			c := newRESTClient()
			return runCallOK(func() (*client.Response, error) {
				return c.ReplaceOrder(context.Background(), args[0], args[1], spec)
			})
		},
	}
	orderSpecFlags(replaceOrderCmd, &replaceFile, &replaceJSON)
	replaceOrderCmd.Flags().BoolVar(&replaceExecute, "execute", false,
		"actually send the replacement (default is dry-run)")
	root.AddCommand(replaceOrderCmd)

	var cancelExecute bool
	cancelOrderCmd := &cobra.Command{
		Use:   "cancel-order <order-id> <account-hash>",
		Short: "Cancel an order — DRY-RUN unless --execute (schwab-py: cancel_order)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/trader/v1/accounts/" + args[1] + "/orders/" + args[0]
			if handled, err := dryRunGuard(cancelExecute, "DELETE", path, nil); handled {
				return err
			}
			if err := requireCredentials(); err != nil {
				return err
			}
			c := newRESTClient()
			return runCallOK(func() (*client.Response, error) {
				return c.CancelOrder(context.Background(), args[0], args[1])
			})
		},
	}
	cancelOrderCmd.Flags().BoolVar(&cancelExecute, "execute", false,
		"actually cancel the order (default is dry-run)")
	root.AddCommand(cancelOrderCmd)

	var previewFile, previewJSON string
	previewOrderCmd := &cobra.Command{
		Use:   "preview-order <account-hash>",
		Short: "Preview an order without placing it (schwab-py: preview_order)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := readOrderSpec(previewFile, previewJSON)
			if err != nil {
				return err
			}
			if err := requireCredentials(); err != nil {
				return err
			}
			c := newRESTClient()
			return runCallOK(func() (*client.Response, error) {
				return c.PreviewOrder(context.Background(), args[0], spec)
			})
		},
	}
	orderSpecFlags(previewOrderCmd, &previewFile, &previewJSON)
	root.AddCommand(previewOrderCmd)

	////////////////////////////////////////////////////////////////////////
	// Transactions

	var txStart, txEnd, txSymbol string
	var txTypes []string
	getTransactionsCmd := &cobra.Command{
		Use:   "get-transactions <account-hash>",
		Short: "Transactions for an account (schwab-py: get_transactions)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := client.TransactionsQuery{}
			if txStart != "" {
				t, err := parseTimeFlag(txStart)
				if err != nil {
					return err
				}
				q.StartDate = &t
			}
			if txEnd != "" {
				t, err := parseTimeFlag(txEnd)
				if err != nil {
					return err
				}
				q.EndDate = &t
			}
			if len(txTypes) > 0 {
				types, err := parseEnumList(txTypes, "types",
					func(s string) (client.TransactionType, error) {
						t := client.TransactionType(s)
						if !t.Valid() {
							return "", fmt.Errorf("invalid value %q", s)
						}
						return t, nil
					})
				if err != nil {
					return err
				}
				q.Types = types
			}
			if txSymbol != "" {
				q.Symbol = &txSymbol
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetTransactions(ctx, args[0], q)
			})
		},
	}
	getTransactionsCmd.Flags().StringVar(&txStart, "start-date", "",
		"only transactions after this date (default: 60 days ago)")
	getTransactionsCmd.Flags().StringVar(&txEnd, "end-date", "",
		"only transactions before this date (default: now)")
	getTransactionsCmd.Flags().StringSliceVar(&txTypes, "types", nil,
		"transaction types to return (default: all types)")
	getTransactionsCmd.Flags().StringVar(&txSymbol, "symbol", "",
		"only transactions for this symbol")
	root.AddCommand(getTransactionsCmd)

	root.AddCommand(&cobra.Command{
		Use:   "get-transaction <account-hash> <transaction-id>",
		Short: "One transaction for an account (schwab-py: get_transaction)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetTransaction(ctx, args[0], args[1])
			})
		},
	})

	////////////////////////////////////////////////////////////////////////
	// User preferences

	root.AddCommand(&cobra.Command{
		Use:   "get-user-preferences",
		Short: "User preferences incl. streaming info (schwab-py: get_user_preferences)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetUserPreferences(ctx)
			})
		},
	})
}

// orderQueryFlags holds the raw flag values shared by the two order-list
// commands.
type orderQueryFlags struct {
	maxResults int
	from       string
	to         string
	status     string
}

func (f *orderQueryFlags) build() (client.OrderQuery, error) {
	q := client.OrderQuery{}
	if f.maxResults != 0 {
		q.MaxResults = &f.maxResults
	}
	if f.from != "" {
		t, err := parseTimeFlag(f.from)
		if err != nil {
			return q, err
		}
		q.FromEnteredTime = &t
	}
	if f.to != "" {
		t, err := parseTimeFlag(f.to)
		if err != nil {
			return q, err
		}
		q.ToEnteredTime = &t
	}
	if f.status != "" {
		s := client.OrderStatus(f.status)
		if !s.Valid() {
			return q, fmt.Errorf("invalid --status value %q", f.status)
		}
		q.Status = &s
	}
	return q, nil
}

// parseEnumList converts raw string flag values into a typed enum slice,
// failing on the first invalid value (schwab-py's EnumEnforcer behavior).
func parseEnumList[T any](vals []string, _ string,
	conv func(string) (T, error)) ([]T, error) {
	out := make([]T, 0, len(vals))
	for _, v := range vals {
		t, err := conv(v)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// runCallOK is runCall for closures that already manage their own context.
func runCallOK(fn func() (*client.Response, error)) error {
	resp, err := fn()
	if err != nil {
		return err
	}
	return printResponse(resp)
}
