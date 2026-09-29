// Hello Fleet is Fleetwood's reference module.
//
// It is deliberately small and deliberately honest about the sandbox: it reads the clock through
// the host's now() rather than Go's time.Now, writes diagnostics through the host's log rather than
// stdout, and touches nothing else. Everything it renders comes back as a PageSpec that core draws
// with its own design system, so no module code runs in Fleetwood's origin.
package main

import (
	"encoding/json"
	"strconv"

	"github.com/extism/go-pdk"
)

//go:wasmimport extism:host/user log
func hostLog(level, message uint64) uint64

//go:wasmimport extism:host/user now
func hostNow() uint64

// The capability host functions all take one JSON request and return one JSON response. Nothing
// here reaches the outside world directly: every one of these is checked and attributed by core.
//
//go:wasmimport extism:host/user kv_get
func hostKVGet(req uint64) uint64

//go:wasmimport extism:host/user kv_set
func hostKVSet(req uint64) uint64

//go:wasmimport extism:host/user sites_current
func hostSitesCurrent(req uint64) uint64

// logf sends one line to core, attributed to this module.
func logf(level, message string) {
	l := pdk.AllocateString(level)
	defer l.Free()
	m := pdk.AllocateString(message)
	defer m.Free()
	hostLog(l.Offset(), m.Offset())
}

// hostErr is the typed failure core returns for things a module is expected to cope with, such as a
// full quota. It arrives inside the response rather than as a crash, so a module can decide.
type hostErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *hostErr) Error() string { return e.Code + ": " + e.Message }

// call performs one capability call and unpacks the envelope into out.
func call(fn func(uint64) uint64, req any, out any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	in := pdk.AllocateBytes(body)
	defer in.Free()

	resp := pdk.FindMemory(fn(in.Offset()))
	if resp.Length() == 0 {
		return &hostErr{Code: "host_unavailable", Message: "the host returned nothing"}
	}
	var env struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error *hostErr        `json:"error"`
	}
	if err := json.Unmarshal(resp.ReadBytes(), &env); err != nil {
		return err
	}
	if !env.OK {
		if env.Error != nil {
			return env.Error
		}
		return &hostErr{Code: "internal_error", Message: "the host refused without saying why"}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

// visitCount records how often this site has opened the page. It exists to exercise module-private
// storage honestly: the value is per site, survives the invocation, and is subject to the quota.
func visitCount() int {
	const key = "page:visits"
	var got struct {
		Found bool   `json:"found"`
		Value []byte `json:"value"`
	}
	if err := call(hostKVGet, map[string]any{"key": key}, &got); err != nil {
		logf("warn", "could not read the visit count: "+err.Error())
		return 0
	}
	n := 0
	if got.Found {
		n, _ = strconv.Atoi(string(got.Value))
	}
	n++
	if err := call(hostKVSet, map[string]any{"key": key, "value": []byte(strconv.Itoa(n))}, nil); err != nil {
		// A full quota is the module's problem to handle, not a reason to fail the page.
		logf("warn", "could not record the visit: "+err.Error())
	}
	return n
}

// invocation is what core tells the module about the call. A module cannot set any of it.
type invocation struct {
	SiteID    string `json:"site_id"`
	SiteName  string `json:"site_name"`
	Principal string `json:"principal"`
	PageID    string `json:"page_id,omitempty"`
	WidgetID  string `json:"widget_id,omitempty"`
	Size      string `json:"size,omitempty"`
	JobType   string `json:"job_type,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
	Targets   []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"targets,omitempty"`
}

type section struct {
	Kind    string     `json:"kind"`
	Title   string     `json:"title,omitempty"`
	Body    string     `json:"body,omitempty"`
	Items   []item     `json:"items,omitempty"`
	Columns []string   `json:"columns,omitempty"`
	Rows    [][]string `json:"rows,omitempty"`
}

type item struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail,omitempty"`
}

type pageSpec struct {
	Title    string    `json:"title"`
	Sections []section `json:"sections"`
}

func input() invocation {
	var in invocation
	_ = json.Unmarshal(pdk.Input(), &in)
	return in
}

func output(v any) int32 {
	body, err := json.Marshal(v)
	if err != nil {
		logf("error", "could not encode a response: "+err.Error())
		return 1
	}
	pdk.Output(body)
	return 0
}

//go:wasmexport page_data
func pageData() int32 {
	in := input()
	logf("info", "rendering the Hello Fleet page for site "+in.SiteID)

	now := hostNow()
	site := in.SiteName
	if site == "" {
		site = in.SiteID
	}
	// Ask core which site this is rather than trusting the name passed in, and count the visit in
	// module-private storage. Both go through host functions, which is the whole point.
	var current struct {
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Timezone string `json:"timezone"`
	}
	if err := call(hostSitesCurrent, map[string]any{}, &current); err != nil {
		logf("warn", "could not read the current site: "+err.Error())
	} else if current.Name != "" {
		site = current.Name
	}
	visits := visitCount()

	return output(pageSpec{
		Title: "Hello Fleet",
		Sections: []section{
			{
				Kind:  "text",
				Title: "This page came from a module",
				Body: "Everything below was produced inside a WebAssembly sandbox and returned to Fleetwood " +
					"as a specification, not as HTML. The module cannot reach the filesystem, the network " +
					"or Fleetwood's own pages.",
			},
			{
				Kind: "stats",
				Items: []item{
					{Label: "Site", Value: site, Detail: "read back through the host's sites.current()"},
					{Label: "Time zone", Value: orDash(current.Timezone), Detail: "the site's own zone, from core"},
					{Label: "Asked by", Value: orDash(in.Principal), Detail: "core sets this; the module cannot"},
					{Label: "Host clock", Value: strconv.FormatUint(now, 10), Detail: "milliseconds, from the host's now()"},
					{Label: "Times opened", Value: strconv.Itoa(visits), Detail: "kept in this module's own storage for this site"},
				},
			},
			{
				Kind:    "table",
				Title:   "What this module declared",
				Columns: []string{"Capability", "Declared as"},
				Rows: [][]string{
					{"Page", "hello (declarative, needs hello:read)"},
					{"Widget", "hello-count (S and M, needs hello:read)"},
					{"Job type", "say-hello (devices, dry run supported, needs hello:execute)"},
					{"Network access", "none"},
					{"Connection privileges", "none"},
				},
			},
		},
	})
}

//go:wasmexport widget_data
func widgetData() int32 {
	in := input()
	return output(map[string]any{
		"kind":   "stat",
		"label":  "Hello Fleet",
		"value":  "ready",
		"detail": "site " + orDash(in.SiteID),
	})
}

//go:wasmexport run_job
func runJob() int32 {
	in := input()
	if in.DryRun {
		logf("info", "dry run: "+strconv.Itoa(len(in.Targets))+" targets would be greeted")
		return output(map[string]any{
			"status":  "ok",
			"dry_run": true,
			"detail":  "would greet " + strconv.Itoa(len(in.Targets)) + " targets",
		})
	}
	for _, t := range in.Targets {
		logf("info", "hello "+orDash(t.Name))
	}
	return output(map[string]any{
		"status": "ok",
		"detail": "greeted " + strconv.Itoa(len(in.Targets)) + " targets",
	})
}

//go:wasmexport on_enable
func onEnable() int32 {
	in := input()
	logf("info", "enabled on site "+in.SiteID)
	return output(map[string]any{"status": "ok"})
}

//go:wasmexport on_disable
func onDisable() int32 {
	in := input()
	logf("info", "disabled on site "+in.SiteID)
	return output(map[string]any{"status": "ok"})
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func main() {}
