// Copyright (C) 2019 Algorand, Inc.
// This file is part of Xchain
//
// Xchain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// Xchain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Xchain.  If not, see <https://www.gnu.org/licenses/>.

package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNodeExporterArgs(t *testing.T) {
	passTestcases := map[string][]string{
		"./node_exporter":                                                                           []string{"./node_exporter", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"},                                               // simple case
		"./node_exporter --collector.systemd":                                                       []string{"./node_exporter", "--collector.systemd", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"},                        // extended case with one argument
		"./node_exporter random --collector.systemd":                                                []string{"./node_exporter", "random", "--collector.systemd", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"},              // extended case multiple arguments
		"/usr/bin/local/node_exporter --collector.systemd random":                                   []string{"/usr/bin/local/node_exporter", "--collector.systemd", "random", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"}, // other executable path
		" /usr/bin/local/node_exporter --collector.systemd random":                                  []string{"/usr/bin/local/node_exporter", "--collector.systemd", "random", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"}, // space at beginning of option
		"./node_exporter --web.telemetry-path=/foobar --web.listen-address=:9090 ":                  []string{"./node_exporter", "--web.listen-address=:9090", "--web.telemetry-path=/foobar"},                                                // overriding defaults
		"./node_exporter --web.listen-address=:8080  --web.telemetry-path=/barfoo":                  []string{"./node_exporter", "--web.listen-address=:8080", "--web.telemetry-path=/barfoo"},                                                // overriding defaults different order and multiple spaces
		"./node_exporter --web.listen-address=:9090  --collector.proc --web.telemetry-path=/foobar": []string{"./node_exporter", "--collector.proc", "--web.listen-address=:9090", "--web.telemetry-path=/foobar"},                            // argument in between the persistent ones
		"./node_exporter --web.listen-address=:9090  --collector.test  --collector.systemd ":        []string{"./node_exporter", "--collector.test", "--collector.systemd", "--web.listen-address=:9090", "--web.telemetry-path=/metrics"},    // argument after persistent one
	}
	for test, expected := range passTestcases {
		vargs := parseNodeExporterArgs(test, ":9100", "/metrics")
		require.Equalf(t, vargs, expected, "Argument parsing did not result in expected value for: %v, got: %v, want: %v.", test, vargs, expected)
	}

	failTestcases := map[string][]string{
		"./node_exporter":                                          []string{"./node_exporter", "--web.listen-address=:9090", "--web.telemetry-path=/foobar"},                                                 // default arguments not being passed
		"./node_exporter --collector.systemd":                      []string{"./node_exporter", "--web.listen-address=:9100", "--web.telemetry-path=/metrics", "--collector.systemd"},                         // incorrect order of persistent and added options
		"./node_exporter random --collector.systemd":               []string{"./node_exporter", "--collector.systemd", "random", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"},               // reversed order of persistent options
		" /usr/bin/local/node_exporter --collector.systemd random": []string{" /usr/bin/local/node_exporter", "--collector.systemd", "random", "--web.listen-address=:9100", "--web.telemetry-path=/metrics"}, // space at beginning of option preserved
	}
	for test, notexpected := range failTestcases {
		vargs := parseNodeExporterArgs(test, ":9100", "/metrics")
		require.NotEqualf(t, vargs, notexpected, "Argument parsing did result in expected value for: %v, got: %v, want: %v.", test, vargs, notexpected)
	}
}
