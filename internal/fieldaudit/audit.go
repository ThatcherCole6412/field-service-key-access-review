package fieldaudit

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Key struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

type Access struct {
	Photos     bool `json:"work_order_photos"`
	Dispatch   bool `json:"dispatch_status"`
	FollowUp   bool `json:"technician_follow_up"`
	FullAccess bool `json:"full_access"`
}

type Finding struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
	Access Access   `json:"access"`
}

type Review struct {
	Date     string    `json:"date"`
	Findings []Finding `json:"findings"`
}

const (
	PhotosScope   = "fieldservice.work_order.photos"
	DispatchScope = "fieldservice.dispatch.status"
	FollowUpScope = "fieldservice.technician.follow_up"
)

func Build(keys []Key, date time.Time) Review {
	findings := make([]Finding, 0, len(keys))
	for _, key := range keys {
		scopes := append([]string(nil), key.Scopes...)
		sort.Strings(scopes)
		access := classify(scopes)
		findings = append(findings, Finding{
			ID: key.ID, Name: key.Name, Scopes: scopes, Access: access,
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Name == findings[j].Name {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].Name < findings[j].Name
	})
	return Review{Date: date.UTC().Format("2006-01-02"), Findings: findings}
}

func classify(scopes []string) Access {
	var access Access
	for _, scope := range scopes {
		switch scope {
		case "*":
			access = Access{Photos: true, Dispatch: true, FollowUp: true, FullAccess: true}
		case PhotosScope:
			access.Photos = true
		case DispatchScope:
			access.Dispatch = true
		case FollowUpScope:
			access.FollowUp = true
		}
	}
	return access
}

func Markdown(review Review) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Field-service access review - %s\n\n", review.Date)
	out.WriteString("| Key | Work-order photos | Dispatch status | Technician follow-up |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	for _, finding := range review.Findings {
		fmt.Fprintf(&out, "| %s | %s | %s | %s |\n",
			cell(finding.Name), mark(finding.Access.Photos), mark(finding.Access.Dispatch), mark(finding.Access.FollowUp))
	}
	return out.String()
}

func mark(value bool) string {
	if value {
		return "allowed"
	}
	return "not granted"
}

func cell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
