/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Summary columns as printed by qrstat 9.0 and 9.1:
// "%7u %-10.10s %-12.12s %-5.5s %-20.20s %-20.20s %8s". The offsets are
// relative to the end of the id column, because an id wider than 7 digits
// shifts all following columns to the right.
const (
	summaryIDWidth     = 7
	summaryNameStart   = 1
	summaryNameWidth   = 10
	summaryOwnerStart  = 12
	summaryOwnerWidth  = 12
	summaryStateStart  = 25
	summaryStateWidth  = 5
	summaryStartStart  = 31
	summaryEndStart    = 52
	summaryDurationPos = 73
	summaryDateWidth   = 20
)

// detailKeyWidth is the width of the attribute name column of qrstat -ar
// ("%-30.30s "). The value starts after one more space.
const detailKeyWidth = 30

// detailSeparator starts each AR block of qrstat -ar.
const detailSeparator = "--------------------------------------------------------------------------------"

// notFoundHeader is the first line qrstat 9.0 and 9.1 print when none of
// the ARs requested with -ar exists.
const notFoundHeader = "Following advance reservations do not exist:"

// Date layouts of qrstat: the summary prints seconds, the detail view
// microseconds. Both are in local time.
var reservationDateLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.000000",
}

// parseReservationTime parses a qrstat date in the local time zone.
func parseReservationTime(s string) (time.Time, error) {
	for _, layout := range reservationDateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised qrstat date %q", s)
}

// parseReservationDuration parses a qrstat duration. 9.0 and 9.1 print
// HH:MM:SS with unbounded hours; 9.2 prints D:HH:MM:SS for a day or more.
func parseReservationDuration(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 && len(parts) != 4 {
		return 0, fmt.Errorf("unrecognised qrstat duration %q", s)
	}
	var total int64
	units := []int64{86400, 3600, 60, 1}[4-len(parts):]
	for i, p := range parts {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("unrecognised qrstat duration %q", s)
		}
		total += n * units[i]
	}
	return time.Duration(total) * time.Second, nil
}

// column returns line[start:end] trimmed of spaces; end < 0 means up to the
// end of the line. Out of range positions yield the empty string.
func column(line string, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end < 0 || end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

// parseSummaryRow parses one AR row of the summary. It returns false if
// the line is not a row (for example an -explain message line).
func parseSummaryRow(line string) (ReservationSummary, bool, error) {
	trimmed := strings.TrimLeft(line, " ")
	digits := 0
	for digits < len(trimmed) && trimmed[digits] >= '0' && trimmed[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits == len(trimmed) || trimmed[digits] != ' ' {
		return ReservationSummary{}, false, nil
	}
	idEnd := len(line) - len(trimmed) + digits
	if idEnd < summaryIDWidth || len(line) <= idEnd+summaryDurationPos {
		return ReservationSummary{}, false, nil
	}
	id, err := strconv.ParseInt(trimmed[:digits], 10, 64)
	if err != nil || id <= 0 {
		return ReservationSummary{}, false, fmt.Errorf("invalid ar id %q", trimmed[:digits])
	}
	at := func(start, width int) string {
		if width < 0 {
			return column(line, idEnd+start, -1)
		}
		return column(line, idEnd+start, idEnd+start+width)
	}
	// qrstat truncates name and owner by bytes, which can split a
	// multi-byte character; drop such a fragment so the result stays valid
	// UTF-8 (and valid JSON).
	s := ReservationSummary{
		ID:    id,
		Name:  strings.ToValidUTF8(at(summaryNameStart, summaryNameWidth), ""),
		Owner: strings.ToValidUTF8(at(summaryOwnerStart, summaryOwnerWidth), ""),
		State: ReservationState(at(summaryStateStart, summaryStateWidth)),
	}
	if s.StartTime, err = parseReservationTime(at(summaryStartStart, summaryDateWidth)); err != nil {
		return ReservationSummary{}, false, err
	}
	if s.EndTime, err = parseReservationTime(at(summaryEndStart, summaryDateWidth)); err != nil {
		return ReservationSummary{}, false, err
	}
	if s.Duration, err = parseReservationDuration(at(summaryDurationPos, -1)); err != nil {
		return ReservationSummary{}, false, err
	}
	return s, true, nil
}

// looksLikeRow reports whether line starts with an id followed by a column
// gap, the way every summary row does.
func looksLikeRow(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	digits := 0
	for digits < len(trimmed) && trimmed[digits] >= '0' && trimmed[digits] <= '9' {
		digits++
	}
	return digits > 0 && strings.HasPrefix(trimmed[digits:], " ")
}

// ParseSummary parses the output of qrstat without -ar (optionally with
// -explain). Empty output yields an empty slice.
func ParseSummary(output string) ([]ReservationSummary, error) {
	reservations := []ReservationSummary{}
	for i, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, " \r")
		if line == "" || strings.HasPrefix(line, "ar-id") || strings.HasPrefix(line, "-----") {
			continue
		}
		row, ok, err := parseSummaryRow(line)
		if err != nil {
			return nil, fmt.Errorf("qrstat line %d: %w", i+1, err)
		}
		if ok {
			reservations = append(reservations, row)
			continue
		}
		// A line which starts like a row but does not parse as one is a
		// changed or truncated format; report it instead of attaching it
		// to the previous AR as a message.
		if looksLikeRow(line) {
			return nil, fmt.Errorf("qrstat line %d: malformed row %q", i+1, line)
		}
		// -explain prints each error reason indented below its AR row.
		if line[0] == ' ' && len(reservations) > 0 {
			last := &reservations[len(reservations)-1]
			last.Messages = append(last.Messages, strings.TrimSpace(line))
			continue
		}
		return nil, fmt.Errorf("qrstat line %d: unexpected line %q", i+1, line)
	}
	return reservations, nil
}

// detailAttribute is one attribute of a qrstat -ar block. Multi-line
// attributes keep one entry in lines per output line.
type detailAttribute struct {
	key   string
	lines []string
}

// splitDetailBlocks splits qrstat -ar output into one attribute list per AR.
func splitDetailBlocks(output string) ([][]detailAttribute, error) {
	var blocks [][]detailAttribute
	for i, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, " \r")
		if line == "" {
			continue
		}
		if line == detailSeparator {
			blocks = append(blocks, nil)
			continue
		}
		if len(blocks) == 0 {
			return nil, fmt.Errorf("qrstat line %d: attribute before the first AR separator: %q", i+1, line)
		}
		key := column(line, 0, detailKeyWidth)
		value := column(line, detailKeyWidth+1, -1)
		current := &blocks[len(blocks)-1]
		if key == "" {
			// A blank key column continues the previous attribute.
			if len(*current) == 0 {
				return nil, fmt.Errorf("qrstat line %d: continuation line without attribute", i+1)
			}
			last := &(*current)[len(*current)-1]
			last.lines = append(last.lines, value)
			continue
		}
		*current = append(*current, detailAttribute{key: key, lines: []string{value}})
	}
	return blocks, nil
}

// splitList splits a comma separated qrstat value. 9.0 and 9.1 separate
// some lists with ", ", 9.2 with ",".
func splitList(s string) []string {
	var items []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// splitTopLevel splits s at commas which are not inside parentheses.
func splitTopLevel(s string) []string {
	var items []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				items = append(items, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(items, strings.TrimSpace(s[start:]))
}

// parseNameValueList parses "a=1, b=2" into a map. Values keep any further
// '=' characters.
func parseNameValueList(s string) (map[string]string, error) {
	m := map[string]string{}
	for _, item := range splitList(s) {
		name, value, ok := strings.Cut(item, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid name=value entry %q", item)
		}
		m[name] = value
	}
	return m, nil
}

// parseQueueSlots parses "all.q@host=2,all.q@other=1".
func parseQueueSlots(s string) (map[string]int, error) {
	pairs, err := parseNameValueList(s)
	if err != nil {
		return nil, err
	}
	slots := make(map[string]int, len(pairs))
	for queue, value := range pairs {
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid slot count in %q", queue+"="+value)
		}
		slots[queue] = n
	}
	return slots, nil
}

// parseGrantedResources parses the granted_resources_list lines, one per
// host, e.g. "node01: gpu=2(gpu0 gpu1)".
func parseGrantedResources(lines []string) (map[string][]GrantedResource, error) {
	result := map[string][]GrantedResource{}
	for _, line := range lines {
		host, rest, ok := strings.Cut(line, ":")
		host = strings.TrimSpace(host)
		if !ok || host == "" {
			return nil, fmt.Errorf("invalid granted resources line %q", line)
		}
		for _, entry := range splitTopLevel(rest) {
			if entry == "" {
				continue
			}
			name, value, ok := strings.Cut(entry, "=")
			if !ok || name == "" {
				return nil, fmt.Errorf("invalid granted resource %q", entry)
			}
			resource := GrantedResource{Name: name, Amount: value}
			if open := strings.IndexByte(value, '('); open >= 0 && strings.HasSuffix(value, ")") {
				resource.Amount = value[:open]
				resource.IDs = strings.Fields(value[open+1 : len(value)-1])
			}
			result[host] = append(result[host], resource)
		}
	}
	return result, nil
}

// applyDetailAttribute stores one attribute in r.
func applyDetailAttribute(r *Reservation, a detailAttribute) error {
	value := a.lines[0]
	var err error
	switch a.key {
	case "id":
		r.ID, err = strconv.ParseInt(value, 10, 64)
	case "name":
		r.Name = value
	case "owner":
		r.Owner = value
	case "state":
		r.State = ReservationState(value)
	case "start_time":
		r.StartTime, err = parseReservationTime(value)
	case "end_time":
		r.EndTime, err = parseReservationTime(value)
	case "submission_time":
		r.SubmissionTime, err = parseReservationTime(value)
	case "duration":
		r.Duration, err = parseReservationDuration(value)
	case "message":
		r.Messages = append(r.Messages, a.lines...)
	case "group":
		r.Group = value
	case "account":
		r.Account = value
	case "resource_list":
		r.ResourceList, err = parseNameValueList(value)
	case "error_handling":
		r.HardErrorHandling = value == "true"
	case "exec_queue_list", "granted_slots_list":
		r.ExecQueueList, err = parseQueueSlots(value)
	case "granted_parallel_environment":
		name, slotRange, ok := strings.Cut(value, " slots ")
		if !ok {
			return fmt.Errorf("invalid granted_parallel_environment %q", value)
		}
		r.GrantedParallelEnvironment = &GrantedPE{Name: name, Range: slotRange}
	case "master hard queue_list":
		r.MasterQueueList = splitList(value)
	case "checkpoint_name":
		r.CheckpointName = value
	case "mail_options":
		r.MailOptions = value
	case "mail_list":
		r.MailList = splitList(value)
	case "acl_list":
		r.ACLList = splitList(value)
	case "xacl_list":
		r.XACLList = splitList(value)
	case "binding":
		r.Binding = value
	case "exec_binding_list":
		r.ExecBindingList, err = parseNameValueList(strings.Join(a.lines, ","))
	case "granted_resources_list":
		r.GrantedResourcesList, err = parseGrantedResources(a.lines)
	default:
		if r.ExtraFields == nil {
			r.ExtraFields = map[string]string{}
		}
		r.ExtraFields[a.key] = strings.Join(a.lines, "\n")
	}
	if err != nil {
		return fmt.Errorf("attribute %s: %w", a.key, err)
	}
	return nil
}

// ParseDetail parses the output of qrstat -ar for one or more ARs.
func ParseDetail(output string) ([]Reservation, error) {
	blocks, err := splitDetailBlocks(output)
	if err != nil {
		return nil, err
	}
	reservations := make([]Reservation, 0, len(blocks))
	for _, block := range blocks {
		var r Reservation
		seen := map[string]bool{}
		for _, attribute := range block {
			// Attributes other than message appear at most once per AR. A
			// repeated one would silently replace the earlier value.
			if attribute.key != "message" {
				if seen[attribute.key] {
					return nil, fmt.Errorf("qrstat ar %d: attribute %s appears twice", r.ID, attribute.key)
				}
				seen[attribute.key] = true
			}
			if err := applyDetailAttribute(&r, attribute); err != nil {
				return nil, fmt.Errorf("qrstat ar %d: %w", r.ID, err)
			}
		}
		if r.ID <= 0 {
			return nil, fmt.Errorf("qrstat: AR block without a valid id")
		}
		reservations = append(reservations, r)
	}
	return reservations, nil
}
