package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexklibisz/terrifi/internal/unifi"
)

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

// TestClientsEntryValues_basicShape exercises the model→object conversion
// against a representative *unifi.Client populated with the read-only fields
// the controller emits on /rest/user. We do not exhaustively re-test the
// existing Client fields (those are covered in client_device_resource_test.go);
// the focus here is the new enrichment fields and the null/empty handling.
func TestClientsEntryValues_basicShape(t *testing.T) {
	blocked := true
	c := &unifi.Client{
		ID:                        "abc123",
		MAC:                       "aa:bb:cc:dd:ee:ff",
		Name:                      "alias",
		Hostname:                  "device.local",
		LastIP:                    "192.168.10.42",
		FixedIP:                   "192.168.10.50",
		NetworkID:                 "net1",
		LastConnectionNetworkName: "10 Core",
		IsWired:                   true,
		IsGuest:                   false,
		OUI:                       "Apple, Inc.",
		Blocked:                   &blocked,
		Note:                      "lab note",
		LocalDNSRecord:            "dev.lab.example",
		FixedApMAC:                "11:22:33:44:55:66",
		FirstSeen:                 1700000000,
		LastSeen:                  1700000500,
	}

	v := clientsEntryValues(c)

	assert.Equal(t, "abc123", v["id"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", v["mac"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "alias", v["name"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "device.local", v["hostname"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "192.168.10.42", v["ip"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "192.168.10.50", v["fixed_ip"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "net1", v["network_id"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "10 Core", v["network_name"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "Apple, Inc.", v["oui"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "lab note", v["note"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "dev.lab.example", v["local_dns_record"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, "11:22:33:44:55:66", v["fixed_ap_mac"].(interface{ ValueString() string }).ValueString())
	assert.Equal(t, true, v["is_wired"].(interface{ ValueBool() bool }).ValueBool())
	assert.Equal(t, false, v["is_guest"].(interface{ ValueBool() bool }).ValueBool())
	assert.Equal(t, true, v["blocked"].(interface{ ValueBool() bool }).ValueBool())
	assert.Equal(t, int64(1700000000), v["first_seen"].(interface{ ValueInt64() int64 }).ValueInt64())
	assert.Equal(t, int64(1700000500), v["last_seen"].(interface{ ValueInt64() int64 }).ValueInt64())
}

// TestClientsEntryValues_emptyStringsAreNull confirms that absent/empty
// optional strings decode to null on the object value (matches the
// device_data_source convention).
func TestClientsEntryValues_emptyStringsAreNull(t *testing.T) {
	c := &unifi.Client{
		ID:  "minimal",
		MAC: "00:11:22:33:44:55",
	}

	v := clientsEntryValues(c)

	for _, attr := range []string{"name", "hostname", "ip", "fixed_ip", "network_id", "network_name", "oui", "note", "local_dns_record", "fixed_ap_mac"} {
		s := v[attr].(interface{ IsNull() bool })
		assert.True(t, s.IsNull(), "expected %s to be null", attr)
	}
}

// TestUnifiClient_UnmarshalEnrichmentFields verifies that a realistic
// /rest/user response (as observed on UOS 5.1.15) decodes the new read-only
// fields into the expected Go-struct shape.
func TestUnifiClient_UnmarshalEnrichmentFields(t *testing.T) {
	// Trimmed to fields the data source consumes — the full controller payload
	// has many more keys but they are dropped by unifi.Client's struct shape.
	payload := []byte(`{
		"_id": "u1",
		"mac": "c8:2a:14:00:00:01",
		"hostname": "tfacc-host",
		"last_ip": "192.168.10.99",
		"last_connection_network_id": "n1",
		"last_connection_network_name": "10 Core",
		"is_wired": true,
		"is_guest": false,
		"oui": "Apple, Inc.",
		"first_seen": 1780049167,
		"last_seen": 1780049500
	}`)

	var c unifi.Client
	require.NoError(t, json.Unmarshal(payload, &c))

	assert.Equal(t, "u1", c.ID)
	assert.Equal(t, "c8:2a:14:00:00:01", c.MAC)
	assert.Equal(t, "tfacc-host", c.Hostname)
	assert.Equal(t, "192.168.10.99", c.LastIP)
	assert.Equal(t, "n1", c.LastConnectionNetworkID)
	assert.Equal(t, "10 Core", c.LastConnectionNetworkName)
	assert.True(t, c.IsWired)
	assert.False(t, c.IsGuest)
	assert.Equal(t, "Apple, Inc.", c.OUI)
	assert.Equal(t, int64(1780049167), c.FirstSeen)
	assert.Equal(t, int64(1780049500), c.LastSeen)
}

// ---------------------------------------------------------------------------
// Acceptance tests
// ---------------------------------------------------------------------------

// TestAccClientsDataSource_basic creates a known terrifi_client_device, then
// reads the terrifi_clients data source and asserts the new client shows up
// (by MAC) with the expected enrichment fields populated.
//
// We do not assert on the LIST length because the bench may carry leftover
// clients from prior tests. We assert on the specific MAC instead, which is
// stable per-run.
func TestAccClientsDataSource_basic(t *testing.T) {
	mac := randomMAC()
	name := fmt.Sprintf("tfacc-clients-%s", randomSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "terrifi_client_device" "test" {
  mac  = %q
  name = %q
  note = "TestAccClientsDataSource_basic"
}

data "terrifi_clients" "all" {
  depends_on = [terrifi_client_device.test]
}
`, mac, name),
				Check: resource.ComposeTestCheckFunc(
					// site echoes back as a computed default.
					resource.TestCheckResourceAttr("data.terrifi_clients.all", "site", "default"),
					// At minimum the newly-created client must appear.
					resource.TestCheckTypeSetElemNestedAttrs(
						"data.terrifi_clients.all",
						"clients.*",
						map[string]string{
							"mac":  mac,
							"name": name,
							"note": "TestAccClientsDataSource_basic",
						},
					),
				),
			},
		},
	})
}
