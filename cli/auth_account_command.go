// Copyright 2023-2024 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/fisk"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/nats-io/nats-server/v2/server"
	ab "github.com/synadia-io/jwt-auth-builder.go"
)

type authAccountCommand struct {
	accountName          string
	operatorName         string
	expiry               time.Duration
	bearerAllowed        bool
	maxSubs              int64
	maxConns             int64
	maxPayloadString     string
	maxPayload           int64
	maxLeafnodes         int64
	maxImports           int64
	maxExports           int64
	jetStream            bool
	defaults             bool
	maxAckPending        int64
	storeMaxString       string
	storeMax             int64
	storeMaxStreamString string
	storeMaxStream       int64
	memMaxString         string
	memMax               int64
	memMaxStreamString   string
	memMaxStream         int64
	streamSizeRequired   bool
	maxStreams           int64
	maxConsumers         int64
	listNames            bool
	force                bool
	skRole               string
	locale               string
	connTypes            []string
	pubAllow             []string
	pubDeny              []string
	subAllow             []string
	subDeny              []string
	showJWT              bool
}

func configureAuthAccountCommand(auth commandHost) {
	c := &authAccountCommand{}

	// TODO:
	//   - rm is written but builder doesnt remove from disk https://github.com/synadia-io/jwt-auth-builder.go/issues/22
	//	 - edit should diff and prompt
	//   - imports/exports

	acct := auth.Command("account", "Manage NATS Accounts").Hidden().Alias("a").Alias("acct")

	addCreateFlags := func(f *fisk.CmdClause, edit bool) {
		f.Flag("expiry", "How long this account should be valid for as a duration").PlaceHolder("DURATION").DurationVar(&c.expiry)
		f.Flag("bearer", "Allows bearer tokens").Default("false").BoolVar(&c.bearerAllowed)
		f.Flag("subscriptions", "Maximum allowed subscriptions").Default("-1").Int64Var(&c.maxSubs)
		f.Flag("connections", "Maximum allowed connections").Default("-1").Int64Var(&c.maxConns)
		f.Flag("payload", "Maximum allowed payload").PlaceHolder("BYTES").StringVar(&c.maxPayloadString)
		f.Flag("leafnodes", "Maximum allowed Leafnode connections").Default("-1").Int64Var(&c.maxLeafnodes)
		f.Flag("imports", "Maximum allowed imports").Default("-1").Int64Var(&c.maxImports)
		f.Flag("exports", "Maximum allowed exports").Default("-1").Int64Var(&c.maxExports)
		f.Flag("jetstream", "Enables JetStream").Default("false").UnNegatableBoolVar(&c.jetStream)
		f.Flag("js-streams", "Sets the maximum Streams the account can have").Default("-1").Int64Var(&c.maxStreams)
		f.Flag("js-consumers", "Sets the maximum Consumers the account can have").Default("-1").Int64Var(&c.maxConsumers)
		f.Flag("js-disk", "Sets a Disk Storage quota").PlaceHolder("BYTES").StringVar(&c.storeMaxString)
		f.Flag("js-disk-stream", "Sets the maximum size a Disk Storage stream may be").PlaceHolder("BYTES").Default("-1").StringVar(&c.storeMaxStreamString)
		f.Flag("js-memory", "Sets a Memory Storage quota").PlaceHolder("BYTES").StringVar(&c.memMaxString)
		f.Flag("js-memory-stream", "Sets the maximum size a Memory Storage stream may be").PlaceHolder("BYTES").Default("-1").StringVar(&c.memMaxStreamString)
		f.Flag("js-max-pending", "Default Max Ack Pending for Tier 0 limits").PlaceHolder("MESSAGES").Int64Var(&c.maxAckPending)
		f.Flag("js-stream-size-required", "Requires Streams to have a maximum size declared").UnNegatableBoolVar(&c.streamSizeRequired)
	}

	add := acct.Command("add", "Adds a new Account").Action(c.addAction)
	add.Arg("name", "Unique name for this Account").StringVar(&c.accountName)
	add.Flag("operator", "Operator to add the account to").StringVar(&c.operatorName)
	addCreateFlags(add, false)
	add.Flag("defaults", "Accept default values without prompting").UnNegatableBoolVar(&c.defaults)

	info := acct.Command("info", "Show Account information").Alias("i").Alias("show").Alias("view").Action(c.infoAction)
	info.Arg("name", "Account to view").StringVar(&c.accountName)
	info.Flag("operator", "Operator hosting the account").StringVar(&c.operatorName)

	edit := acct.Command("edit", "Edit account settings").Alias("update").Action(c.editAction)
	edit.Arg("name", "Unique name for this Account").StringVar(&c.accountName)
	edit.Flag("operator", "Operator to add the account to").StringVar(&c.operatorName)
	addCreateFlags(edit, false)

	ls := acct.Command("ls", "List accounts").Action(c.lsAction)
	ls.Arg("operator", "Operator to act on").StringVar(&c.operatorName)
	ls.Flag("names", "Show just the Account names").UnNegatableBoolVar(&c.listNames)

	rm := acct.Command("rm", "Removes an account").Action(c.rmAction)
	rm.Arg("name", "Account to view").StringVar(&c.accountName)
	rm.Flag("operator", "Operator hosting the account").StringVar(&c.operatorName)
	rm.Flag("force", "Removes without prompting").Short('f').UnNegatableBoolVar(&c.force)

	push := acct.Command("push", "Push the account to the NATS Resolver").Action(c.pushAction)
	push.Arg("name", "Account to act on").StringVar(&c.accountName)
	push.Flag("operator", "Operator to act on").StringVar(&c.operatorName)
	push.Flag("show", "Show the Account JWT before pushing").UnNegatableBoolVar(&c.showJWT)

	sk := acct.Command("keys", "Manage Scoped Signing Keys").Alias("sk").Alias("s")

	skadd := sk.Command("add", "Adds a signing key").Alias("new").Alias("a").Alias("n").Action(c.skAddAction)
	skadd.Arg("name", "Account to act on").StringVar(&c.accountName)
	skadd.Arg("role", "The role to add a key for").StringVar(&c.skRole)
	skadd.Flag("operator", "Operator to act on").StringVar(&c.operatorName)
	skadd.Flag("subscriptions", "Maximum allowed subscriptions").Default("-1").Int64Var(&c.maxSubs)
	skadd.Flag("payload", "Maximum allowed payload").PlaceHolder("BYTES").StringVar(&c.maxPayloadString)
	skadd.Flag("bearer", "Allows bearer tokens").Default("false").BoolVar(&c.bearerAllowed)
	skadd.Flag("locale", "Locale for the client").StringVar(&c.locale)
	skadd.Flag("connection", "Set the allowed connections (nats, ws, wsleaf, mqtt)").EnumsVar(&c.connTypes, "nats", "ws", "leaf", "wsleaf", "mqtt")
	skadd.Flag("pub-allow", "Sets subjects where publishing is allowed").StringsVar(&c.pubAllow)
	skadd.Flag("pub-deny", "Sets subjects where publishing is allowed").StringsVar(&c.pubDeny)
	skadd.Flag("sub-allow", "Sets subjects where subscribing is allowed").StringsVar(&c.subAllow)
	skadd.Flag("sub-deny", "Sets subjects where subscribing is allowed").StringsVar(&c.subDeny)

	skInfo := sk.Command("info", "Show information for a Scoped Signing Key").Action(c.skInfoAction)
	skInfo.Arg("name", "Account to view").StringVar(&c.accountName)
	skInfo.Arg("key", "The role or key to view").StringVar(&c.skRole)
	skInfo.Flag("operator", "Operator to act on").StringVar(&c.operatorName)

	skls := sk.Command("list", "List Scoped Signing Keys").Alias("ls").Action(c.skListAction)
	skls.Arg("name", "Account to act on").StringVar(&c.accountName)
	skls.Flag("operator", "Operator to act on").StringVar(&c.operatorName)

	skrm := sk.Command("rm", "Remove a scoped signing key").Action(c.skRmAction)
	skrm.Arg("name", "Account to act on").StringVar(&c.accountName)
	skrm.Arg("key", "The key to remove").StringVar(&c.skRole)
	skrm.Flag("operator", "Operator to act on").StringVar(&c.operatorName)
	skrm.Flag("force", "Removes without prompting").Short('f').UnNegatableBoolVar(&c.force)
}

func (c *authAccountCommand) selectAccount(pick bool) (*ab.AuthImpl, ab.Operator, ab.Account, error) {
	auth, oper, acct, err := selectOperatorAccount(c.operatorName, c.accountName, pick)
	if err != nil {
		return nil, nil, nil, err
	}

	c.operatorName = oper.Name()
	c.accountName = acct.Name()

	return auth, oper, acct, nil
}

func (c *authAccountCommand) selectOperator(pick bool) (*ab.AuthImpl, ab.Operator, error) {
	auth, oper, err := selectOperator(c.operatorName, pick)
	if err != nil {
		return nil, nil, err
	}

	c.operatorName = oper.Name()

	return auth, oper, err
}

func (c *authAccountCommand) pushAction(_ *fisk.ParseContext) error {
	_, _, acct, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	if c.showJWT {
		fmt.Printf("Account JWT for %s\n", c.accountName)
		fmt.Println()
		fmt.Println(acct.JWT())
		fmt.Println()
	}

	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	expect, _ := currentActiveServers(nc)
	if expect > 0 {
		fmt.Printf("Updating account %s (%s) on %d server(s)\n", acct.Name(), acct.Subject(), expect)
	} else {
		fmt.Printf("Updating Account %s (%s) on all servers\n", acct.Name(), acct.Subject())
	}
	fmt.Println()

	errStr := color.RedString("X")
	okStr := color.GreenString("✓")
	updated := 0
	failed := 0

	subj := fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", acct.Subject())
	err = doReqAsync(acct.JWT(), subj, expect, nc, func(msg []byte) {
		update := server.ServerAPIClaimUpdateResponse{}
		err = json.Unmarshal(msg, &update)
		if err != nil {
			fmt.Printf("%s Invalid JSON response received: %v: %s\n", errStr, err, string(msg))
			failed++
			return
		}

		if update.Error != nil {
			fmt.Printf("%s Update failed on %s: %v\n", errStr, update.Server.Name, update.Error.Description)
			failed++
			return
		}

		fmt.Printf("%s Update completed on %s\n", okStr, update.Server.Name)
		updated++
	})
	if err != nil {
		return err
	}

	fmt.Println()
	if expect > 0 {
		fmt.Printf("Success %d Failed %d Expected %d\n", updated, failed, expect)
	} else {
		fmt.Printf("Success %d Failed %d\n", updated, failed)
	}

	if failed > 0 {
		if expect > 0 {
			return fmt.Errorf("update failed on %d/%d servers", failed, expect)
		}
		return fmt.Errorf("update failed on %d servers", failed)
	}

	if updated == 0 {
		return fmt.Errorf("no servers were updated")
	}

	if expect > 0 && (updated+failed != expect) {
		return fmt.Errorf("received updated from only %d out of %d servers", updated+failed, expect)
	}

	return nil
}

func (c *authAccountCommand) skRmAction(_ *fisk.ParseContext) error {
	_, _, acct, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	if c.skRole == "" {
		err := askOne(&survey.Input{
			Message: "Scoped Signing Key",
			Help:    "The key to remove",
		}, &c.skRole, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if !c.force {
		ok, err := askConfirmation(fmt.Sprintf("Really remove the Scoped Signing Key %s", c.skRole), false)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}
	}

	ok, err := acct.ScopedSigningKeys().Delete(c.skRole)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("key %q not found", c.skRole)
	}

	fmt.Printf("key %q removed\n", c.skRole)
	return nil
}

func (c *authAccountCommand) skInfoAction(_ *fisk.ParseContext) error {
	_, _, acct, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	if c.skRole == "" {
		err := askOne(&survey.Input{
			Message: "Role or Key",
			Help:    "The key to view by role or key name",
		}, &c.skRole, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	var ok bool
	sk := acct.ScopedSigningKeys().GetScopeByRole(c.skRole)
	if sk == nil {
		sk, ok = acct.ScopedSigningKeys().GetScope(c.skRole)
		if !ok {
			return fmt.Errorf("no role or scope found matching %q", c.skRole)
		}
	}

	out, err := c.showSk(sk)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

func (c *authAccountCommand) skAddAction(_ *fisk.ParseContext) error {
	auth, _, acct, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	if c.skRole == "" {
		err := askOne(&survey.Input{
			Message: "Role Name",
			Help:    "The role to associate with this key",
		}, &c.skRole, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if c.maxPayloadString != "" {
		c.maxPayload, err = parseStringAsBytes(c.maxPayloadString)
		if err != nil {
			return err
		}
	}

	scope, err := acct.ScopedSigningKeys().AddScope(c.skRole)
	if err != nil {
		return err
	}

	limits := scope.(userLimitsManager).UserPermissionLimits()
	limits.Subs = c.maxSubs
	limits.Payload = c.maxPayload
	limits.BearerToken = c.bearerAllowed
	limits.Locale = c.locale
	limits.Pub.Allow = c.pubAllow
	limits.Pub.Deny = c.pubDeny
	limits.Sub.Allow = c.subAllow
	limits.Sub.Deny = c.subDeny
	if len(c.connTypes) > 0 {
		limits.AllowedConnectionTypes = c.connectionTypes()
	}

	err = scope.(userLimitsManager).SetUserPermissionLimits(limits)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return c.fShowSk(os.Stdout, scope)
}

func (c *authAccountCommand) fShowSk(w io.Writer, limits ab.ScopeLimits) error {
	out, err := c.showSk(limits)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, out)

	return err
}

func (c *authAccountCommand) showSk(limits ab.ScopeLimits) (string, error) {
	cols := newColumns("Scoped Signing Key %s", limits.Key())

	cols.AddSectionTitle("Config")

	cols.AddRow("Key", limits.Key())
	cols.AddRow("Role", limits.Role())

	err := renderUserLimits(limits, cols)
	if err != nil {
		return "", err
	}

	return cols.Render()
}

func (c *authAccountCommand) connectionTypes() []string {
	var types []string

	for _, t := range c.connTypes {
		switch t {
		case "nats":
			types = append(types, "STANDARD")
		case "ws":
			types = append(types, "WEBSOCKET")
		case "wsleaf":
			types = append(types, "LEAFNODE_WS")
		case "leaf":
			types = append(types, "LEAFNODE")
		case "mqtt":
			types = append(types, "MQTT")
		}
	}

	return types
}

func (c *authAccountCommand) skListAction(_ *fisk.ParseContext) error {
	_, _, acct, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	var table *tbl

	if len(acct.ScopedSigningKeys().List()) > 0 {
		table = newTableWriter("Scoped Signing Keys")
		table.AddHeaders("Name", "Role", "Key", "Pub Perms", "Sub Perms")
		for _, sk := range acct.ScopedSigningKeys().List() {
			scope, _ := acct.ScopedSigningKeys().GetScope(sk)

			pubs := len(scope.PubPermissions().Allow()) + len(scope.PubPermissions().Deny())
			subs := len(scope.SubPermissions().Allow()) + len(scope.SubPermissions().Deny())

			table.AddRow(sk, scope.Role(), scope.Key(), scope.MaxSubscriptions(), pubs, subs)
		}
		fmt.Println(table.Render())
		fmt.Println()
	}

	if len(acct.ScopedSigningKeys().ListRoles()) > 0 {
		table = newTableWriter("Roles")
		table.AddHeaders("Name", "Key")
		for _, r := range acct.ScopedSigningKeys().ListRoles() {
			role := acct.ScopedSigningKeys().GetScopeByRole(r)
			if role == nil {
				continue
			}

			table.AddRow(role.Role(), role.Key())
		}
		fmt.Println(table.Render())
	}

	if table == nil {
		fmt.Println("No Scoped Signing Keys or Roles defined")
	}

	return nil
}

func (c *authAccountCommand) editAction(_ *fisk.ParseContext) error {
	auth, operator, acct, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	// TODO: need to think if we should support disabling jetstream here, possibly by turning
	// --jetstream into a bool and adding an isSet variable, then we could disable it
	err = c.updateAccount(acct, c.jetStream || acct.Limits().JetStream().IsJetStreamEnabled())
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return c.fShowAccount(os.Stdout, operator, acct)
}

func (c *authAccountCommand) rmAction(_ *fisk.ParseContext) error {
	fmt.Println("WARNING: At present deleting is not supported by the nsc store")
	fmt.Println()

	auth, operator, account, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	if !c.force {
		ok, err := askConfirmation(fmt.Sprintf("Really remove the Accouint %s", c.accountName), false)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}
	}

	err = operator.Accounts().Delete(account.Name())
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	fmt.Printf("Removed account %s\n", account.Name())
	return nil
}

func (c *authAccountCommand) lsAction(_ *fisk.ParseContext) error {
	_, operator, err := c.selectOperator(true)
	if err != nil {
		return err
	}

	list := operator.Accounts().List()
	if len(list) == 0 {
		fmt.Println("No Accounts found")
		return nil
	}

	if c.listNames {
		for _, op := range list {
			fmt.Println(op.Name())
		}
		return nil
	}

	table := newTableWriter("Accounts")
	table.AddHeaders("Name", "Subject", "Users", "JetStream", "System")
	for _, acct := range list {
		system := ""
		js := ""
		if acct.Subject() == operator.SystemAccount().Subject() {
			system = "true"
		}
		if acct.Limits().JetStream().IsJetStreamEnabled() {
			js = "true"
		}

		table.AddRow(acct.Name(), acct.Subject(), len(acct.Users().List()), js, system)
	}
	fmt.Println(table.Render())

	return nil
}

func (c *authAccountCommand) infoAction(_ *fisk.ParseContext) error {
	_, operator, account, err := c.selectAccount(true)
	if err != nil {
		return err
	}

	return c.fShowAccount(os.Stdout, operator, account)
}

func (c *authAccountCommand) updateAccount(acct ab.Account, js bool) error {
	limits := acct.Limits().(operatorLimitsManager).OperatorLimits()
	limits.Conn = c.maxConns
	limits.Subs = c.maxSubs
	limits.Payload = c.maxPayload
	limits.LeafNodeConn = c.maxLeafnodes
	limits.Exports = c.maxExports
	limits.Imports = c.maxImports
	limits.DisallowBearer = !c.bearerAllowed

	if js {
		if c.storeMaxStream > 0 {
			limits.JetStreamLimits.DiskMaxStreamBytes = c.storeMaxStream
		}
		if c.memMaxStream > 0 {
			limits.JetStreamLimits.MemoryMaxStreamBytes = c.memMaxStream
		}
		limits.JetStreamLimits.DiskStorage = c.storeMax
		limits.JetStreamLimits.MemoryStorage = c.memMax
		limits.JetStreamLimits.MaxBytesRequired = c.streamSizeRequired
		limits.JetStreamLimits.Consumer = c.maxConsumers
		limits.JetStreamLimits.Streams = c.maxStreams
		limits.JetStreamLimits.MaxAckPending = c.maxAckPending
	}

	err := acct.Limits().(operatorLimitsManager).SetOperatorLimits(limits)
	if err != nil {
		return err
	}

	if c.expiry > 0 {
		err = acct.SetExpiry(time.Now().Add(c.expiry).Unix())
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *authAccountCommand) addAction(_ *fisk.ParseContext) error {
	auth, operator, err := c.selectOperator(true)
	if err != nil {
		return err
	}

	if c.accountName == "" {
		err := askOne(&survey.Input{
			Message: "Account Name",
			Help:    "A unique name for the Account being added",
		}, &c.accountName, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if isAuthItemKnown(operator.Accounts().List(), c.accountName) {
		return fmt.Errorf("account %s already exist", c.accountName)
	}

	acct, err := operator.Accounts().Add(c.accountName)
	if err != nil {
		return err
	}

	if !c.defaults {
		if c.maxConns == -1 {
			c.maxConns, err = askOneInt("Maximum Connections", "-1", "The maximum amount of client connections allowed for this account, set using --connections")
			if err != nil {
				return err
			}
		}

		if c.maxSubs == -1 {
			c.maxSubs, err = askOneInt("Maximum Subscriptions", "-1", "The maximum amount of subscriptions allowed for this account, set using --subscriptions")
			if err != nil {
				return err
			}
		}

		if c.maxPayloadString == "" {
			c.maxPayload, err = askOneBytes("Maximum Message Payload", "-1", "The maximum size any message may have, set using --payload", "")
			if err != nil {
				return err
			}
			c.maxPayloadString = ""
		}

		fmt.Println()
	}

	if c.maxPayloadString != "" {
		c.maxPayload, err = parseStringAsBytes(c.maxPayloadString)
		if err != nil {
			return err
		}
	}

	if c.jetStream {
		if c.storeMaxString == "" {
			c.storeMax, err = askOneBytes("Maximum JetStream Disk Storage", "1GB", "Maximum amount of disk this account may use, set using --js-disk", "JetStream requires maximum Disk usage set")
			if err != nil {
				return err
			}
		}
		if c.memMaxString == "" {
			c.memMax, err = askOneBytes("Maximum JetStream Memory Storage", "1GB", "Maximum amount of memory this account may use, set using --js-memory", "JetStream requires maximum Memory usage set")
			if err != nil {
				return err
			}
		}

		if c.storeMaxString != "" {
			c.storeMax, err = parseStringAsBytes(c.storeMaxString)
			if err != nil {
				return err
			}
		}
		if c.memMaxString != "" {
			c.memMax, err = parseStringAsBytes(c.memMaxString)
			if err != nil {
				return err
			}
		}

		if c.memMaxStreamString != "-1" {
			c.memMaxStream, err = parseStringAsBytes(c.memMaxStreamString)
			if err != nil {
				return err
			}
		}
		if c.storeMaxStreamString != "-1" {
			c.storeMaxStream, err = parseStringAsBytes(c.storeMaxStreamString)
			if err != nil {
				return err
			}
		}
	}

	err = c.updateAccount(acct, c.jetStream)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return c.fShowAccount(os.Stdout, operator, acct)
}

func (c *authAccountCommand) fShowAccount(w io.Writer, operator ab.Operator, acct ab.Account) error {
	out, err := c.showAccount(operator, acct)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, out)

	return err
}

func (c *authAccountCommand) showAccount(operator ab.Operator, acct ab.Account) (string, error) {
	limits := acct.Limits()
	js := limits.JetStream()

	cols := newColumns("Account %s (%s)", acct.Name(), acct.Subject())
	cols.AddSectionTitle("Configuration")
	cols.AddRow("Name", acct.Name())
	cols.AddRow("Account", operator.Name())
	cols.AddRow("Issuer", acct.Issuer())
	cols.AddRow("System Account", operator.SystemAccount().Subject() == acct.Subject())
	cols.AddRow("JetStream", js.IsJetStreamEnabled())
	cols.AddRowIf("Expiry", time.Unix(acct.Expiry(), 0), acct.Expiry() > 0)
	cols.AddRow("Users", len(acct.Users().List()))
	cols.AddSectionTitle("Limits")
	cols.AddRow("Bearer Tokens Allowed", !limits.DisallowBearerTokens())
	cols.AddRowUnlimited("Subscriptions", limits.MaxSubscriptions(), -1)
	cols.AddRowUnlimited("Connections", limits.MaxConnections(), -1)
	cols.AddRowUnlimitedIf("Maximum Payload", humanize.IBytes(uint64(limits.MaxPayload())), limits.MaxPayload() <= 0)
	if limits.MaxData() > 0 {
		cols.AddRow("Data", limits.MaxData()) // only showing when set as afaik its a ngs thing
	}
	cols.AddRowUnlimited("Leafnodes", limits.MaxLeafNodeConnections(), -1)
	cols.AddRowUnlimited("Imports", limits.MaxImports(), -1)
	cols.AddRowUnlimited("Exports", limits.MaxExports(), -1)

	if js.IsJetStreamEnabled() {
		cols.Indent(2)
		cols.AddSectionTitle("JetStream Limits")

		tiers := c.validTiers(acct)

		cols.Indent(4)
		for _, tc := range tiers {
			tier, _ := js.Get(tc)
			if tier == nil {
				continue
			}

			if tc == 0 {
				cols.AddSectionTitle("Account Default Limits")
			} else {
				cols.AddSectionTitle("Tier %d", tc)
			}

			if unlimited, _ := tier.IsUnlimited(); unlimited {
				cols.Indent(6)
				cols.Println("Unlimited")
				cols.Indent(4)
				continue
			}

			maxAck, _ := tier.MaxAckPending()
			maxMem, _ := tier.MaxMemoryStorage()
			maxMemStream, _ := tier.MaxMemoryStreamSize()
			maxConns, _ := tier.MaxConsumers()
			maxDisk, _ := tier.MaxDiskStorage()
			maxDiskStream, _ := tier.MaxDiskStreamSize()
			streams, _ := tier.MaxStreams()
			streamSizeRequired, _ := tier.MaxStreamSizeRequired()

			cols.AddRowUnlimited("Max Ack Pending", maxAck, 0)
			cols.AddRowUnlimited("Maximum Streams", streams, -1)
			cols.AddRowUnlimited("Max Consumers", maxConns, -1)
			cols.AddRow("Max Stream Size Required", streamSizeRequired)
			cols.AddRow("Max File Storage", humanize.IBytes(uint64(maxDisk)))
			cols.AddRowIf("Max File Storage Stream Size", humanize.IBytes(uint64(maxDiskStream)), maxDiskStream > 0)
			cols.AddRow("Max Memory Storage", humanize.IBytes(uint64(maxMem)))
			cols.AddRowIf("Max Memory Storage Stream Size", humanize.IBytes(uint64(maxMemStream)), maxMemStream > 0)
		}

		cols.Indent(0)
	}

	return cols.Render()
}

func (c *authAccountCommand) validTiers(acct ab.Account) []int8 {
	tiers := []int8{}
	for i := int8(0); i <= 5; i++ {
		tier, _ := acct.Limits().JetStream().Get(i)
		if tier != nil {
			tiers = append(tiers, i)
		}
	}

	if len(tiers) > 1 {
		tiers = tiers[1:]
	}

	return tiers
}
