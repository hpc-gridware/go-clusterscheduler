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

// Helpers for the ExtraFields preservation contract.
//
// Every per-object Config struct in this package carries an
// ExtraFields map[string]string. Parsers populate it from the default
// branch of their key-dispatch switch via CaptureExtraField. Modify
// functions emit it (sorted, after typed fields) via WriteExtraFields.
// The point is that an admin parameter set on the cluster via qconf
// -m* that this Go struct does not recognise is round-tripped
// untouched on the next modify rather than being reset to its
// default. See GlobalConfig.ExtraFields for the motivating case
// (port_range, OCS/GCS 9.x release notes section 5.1.2).
//
// Contract:
//   - Typed field wins. If ExtraFields contains a key that matches a
//     typed field's json tag, the typed field's value is what gets
//     written; the ExtraFields entry is silently dropped on emit.
//   - Deterministic order. ExtraFields keys are emitted in sorted
//     order so the ETag canonicalisation that the qontrol layer wraps
//     around the typed struct + extras is stable.
//   - Sanitisation on both sides. ExtraFields is part of the JSON wire
//     format, so a map can arrive either from CaptureExtraField
//     (parsed qconf output) or from an untrusted request body. Keys
//     must be lowercase-snake-case and values free of control
//     characters; the parse path drops violations, the emit path
//     rejects them with an error. Without the emit-side check a value
//     containing a line feed would split into a second directive line
//     and inject an arbitrary attribute into the qconf -M* file.

package core

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// validExtrasKey constrains unrecognised qconf keys to the
// lowercase-snake-case shape that every OCS/GCS parameter has used in
// practice for as long as the format has existed. Anything else --
// keys with backslash, leading `#` (which qconf treats as a comment
// line), `=`, whitespace, or non-ASCII -- is dropped at capture so the
// modify-side temp file cannot be tricked into emitting a corrupted
// directive line. False negatives (a future qconf version introducing
// uppercase or dotted keys) trade safety for forward compatibility;
// adjust the pattern when that happens.
var validExtrasKey = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// CaptureExtraField records an unrecognised key/value pair into
// extras. Intended as the default branch of a Parse*/Show* per-key
// switch.
//
// Lazily initialises the map. Lines that are themselves continuations
// of a preceding multi-line known value (indented by two spaces) are
// skipped -- the outer Show* loop visits them but they belong to a
// known key already consumed.
//
// Backslash-continued unknown values are deliberately NOT captured.
// ParseMultiLineValue concatenates continuation lines without
// preserving the original delimiter (space vs comma vs none), so a
// round trip would silently corrupt the value on re-emit. Rather than
// store a malformed value, the helper drops the key entirely. The
// drop is silent; a future revision should surface it through a
// logger but the qconf package today does not depend on klog. See
// qontrol/todos/185-* for the long-term fix.
func CaptureExtraField(extras *map[string]string, lines []string, i int) {
	line := lines[i]
	if strings.HasPrefix(line, "  ") {
		return
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	key := fields[0]
	if !validExtrasKey.MatchString(key) {
		return
	}
	value, multiline := ParseMultiLineValue(lines, i)
	if multiline {
		return
	}
	value = strings.TrimSpace(value)
	sanitised, ok := SanitizeExtraValue(value)
	if !ok {
		return
	}
	if *extras == nil {
		*extras = map[string]string{}
	}
	(*extras)[key] = sanitised
}

// SanitizeExtraValue rejects values containing characters that would
// corrupt the line-oriented qconf temp file format on the way out.
// All C0 control characters (0x00-0x1F) and DEL (0x7F) are rejected:
// CR/LF would split one key into two lines on re-emit, NUL would
// truncate the line as it passes through libc, tab would break the
// strings.Fields tokenisation in the parser, and other control chars
// signal either a buggy qmaster output or a corrupted persistence
// path. Returns (cleanValue, true) on accept, ("", false) on reject.
func SanitizeExtraValue(v string) (string, bool) {
	for _, r := range v {
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}
	return v, true
}

// WriteExtraFields emits extras to w in sorted key order as
// "key value\n" lines. typedKeys names the attribute names the caller
// has already emitted for the typed fields of the same object (plus
// "extra_fields" itself, which is a JSON carrier key and never a qconf
// attribute); any extras entry whose key collides is dropped (typed
// field wins). Safe to call with empty extras.
//
// Keys and values are validated here rather than trusted from the
// caller: ExtraFields is serialisable (json:"extra_fields"), so a map
// reaching this function may have come from an untrusted request body
// rather than from CaptureExtraField, which is the only other place
// the shape is enforced. An unchecked value containing a line feed
// would split into a second directive line and inject an arbitrary
// attribute into the qconf -M* file. Violations fail loudly instead of
// being skipped, so a client cannot have input silently ignored.
func WriteExtraFields(w io.Writer, extras map[string]string, typedKeys map[string]struct{}) error {
	if len(extras) == 0 {
		return nil
	}
	keys := make([]string, 0, len(extras))
	for k := range extras {
		if _, dup := typedKeys[k]; dup {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !validExtrasKey.MatchString(k) {
			return fmt.Errorf("invalid extra field name %q: must match %s",
				k, validExtrasKey.String())
		}
		v, ok := SanitizeExtraValue(extras[k])
		if !ok {
			return fmt.Errorf("invalid extra field value for %q: contains control characters", k)
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}

// PromoteFromExtras moves a single string-valued entry from extras
// into the typed destination pointer and deletes the source entry, if
// present. No-op when the key is absent.
//
// Intended for v-package ShowGlobalConfiguration (and similar)
// callers that wrap the core parser: the core parser captures every
// non-v9.0 key into ExtraFields, and the v-package then promotes the
// keys it knows about into typed fields. Without the delete the key
// would round-trip twice -- once as a typed field, once as an extras
// entry -- which today is masked by the WriteExtraFields collision
// check but is a footgun to leave open.
//
// Non-string fields (maps, parsed shapes) keep their explicit
// promotion blocks; only the trivial string -> string case collapses
// to this helper.
func PromoteFromExtras(extras map[string]string, key string, dst *string) {
	if v, ok := extras[key]; ok {
		*dst = v
		delete(extras, key)
	}
}

// TypedKeysOf returns the set of attribute names reserved by v's typed
// fields. Empty tags and `-` are skipped. Embedded structs are recursed
// into so v9.1.GlobalConfig (which embeds core.GlobalConfig) yields the
// union of core and v9.1 keys.
//
// The set intentionally includes "extra_fields" itself: that is the
// JSON carrier key for the map, never a qconf attribute, so an extras
// entry keyed "extra_fields" is reserved rather than emitted. Note the
// set is therefore not identical to what the Modify* writers emit --
// they skip the ExtraFields field by Go field name.
//
// Used by the reflection-loop Modify* implementations to build the
// collision-set passed to WriteExtraFields, and reusable by Modify*
// implementations that do explicit field-by-field writes (they can
// pass &cfg to derive the same set without hard-coding the key list).
func TypedKeysOf(v any) map[string]struct{} {
	out := map[string]struct{}{}
	addTypedKeys(reflect.TypeOf(v), out)
	return out
}

func addTypedKeys(rt reflect.Type, out map[string]struct{}) {
	if rt == nil {
		return
	}
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Anonymous {
			addTypedKeys(f.Type, out)
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		if comma := strings.Index(tag, ","); comma >= 0 {
			tag = tag[:comma]
		}
		out[tag] = struct{}{}
	}
}
