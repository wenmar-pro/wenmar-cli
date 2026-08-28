package cmd

import (
	"fmt"
	"net/http"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
)

// breadcrumb builds a single actionable navigation hint.
func breadcrumb(action, cmd string) output.Breadcrumb {
	return output.Breadcrumb{Action: action, Cmd: cmd}
}

// listBreadcrumbs returns actionable next steps for a list command.
func listBreadcrumbs(resource string) []output.Breadcrumb {
	return []output.Breadcrumb{
		breadcrumb("show", fmt.Sprintf("wenmar %s show <id>", resource)),
		breadcrumb("create", fmt.Sprintf("wenmar %s create", resource)),
	}
}

// showBreadcrumbs returns actionable next steps for a show command.
func showBreadcrumbs(resource, id string) []output.Breadcrumb {
	return []output.Breadcrumb{
		breadcrumb("update", fmt.Sprintf("wenmar %s update %s", resource, id)),
		breadcrumb("delete", fmt.Sprintf("wenmar %s delete %s --dry-run", resource, id)),
	}
}

// createBreadcrumbs returns actionable next steps after a create.
func createBreadcrumbs(resource, id string) []output.Breadcrumb {
	return []output.Breadcrumb{
		breadcrumb("show", fmt.Sprintf("wenmar %s show %s", resource, id)),
	}
}

// checkTruncated returns an error if the response was truncated and
// --allow-partial is not set. If --allow-partial is set, it returns a notice
// string describing the truncation.
func checkTruncated(truncated bool, detail string) (string, error) {
	if !truncated {
		return "", nil
	}
	if allowPartial {
		return detail, nil
	}
	return "", &errors.PartialError{Message: "Response was truncated. Use --allow-partial to accept partial data."}
}

// checkTruncatedResponse inspects the X-Wenmar-Truncated header on an HTTP
// response and returns a notice/error via checkTruncated.
func checkTruncatedResponse(hr *http.Response, detail string) (string, error) {
	truncated := false
	if hr != nil && hr.Header.Get("X-Wenmar-Truncated") == "true" {
		truncated = true
	}
	return checkTruncated(truncated, detail)
}
