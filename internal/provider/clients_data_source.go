package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/alexklibisz/terrifi/internal/unifi"
)

var _ datasource.DataSource = &clientsDataSource{}

func NewClientsDataSource() datasource.DataSource {
	return &clientsDataSource{}
}

type clientsDataSource struct {
	client *Client
}

type clientsDataSourceModel struct {
	Site    types.String `tfsdk:"site"`
	Clients types.List   `tfsdk:"clients"`
}

// clientsEntryAttrTypes is the per-entry attribute type map. Kept in sync with
// clientsEntrySchemaAttributes and clientsEntryValues — adding a field means
// touching all three. The set is the intersection of fields the controller
// populates on /rest/user (configured + remembered clients), so a single API
// call hydrates everything.
func clientsEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":               types.StringType,
		"mac":              types.StringType,
		"name":             types.StringType,
		"hostname":         types.StringType,
		"ip":               types.StringType,
		"fixed_ip":         types.StringType,
		"network_id":       types.StringType,
		"network_name":     types.StringType,
		"is_wired":         types.BoolType,
		"is_guest":         types.BoolType,
		"oui":              types.StringType,
		"blocked":          types.BoolType,
		"note":             types.StringType,
		"local_dns_record": types.StringType,
		"fixed_ap_mac":     types.StringType,
		"first_seen":       types.Int64Type,
		"last_seen":        types.Int64Type,
	}
}

func clientsEntrySchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The controller-assigned ID of the client record.",
			Computed:            true,
		},
		"mac": schema.StringAttribute{
			MarkdownDescription: "The MAC address of the client.",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The user-assigned alias for the client (the `name` field on `terrifi_client_device`). Often null for clients that have never had an alias set.",
			Computed:            true,
		},
		"hostname": schema.StringAttribute{
			MarkdownDescription: "The hostname the client reported via DHCP option 12 (if any).",
			Computed:            true,
		},
		"ip": schema.StringAttribute{
			MarkdownDescription: "The IP the client most recently held (controller field `last_ip`). May be null if the controller has never observed the client connecting.",
			Computed:            true,
		},
		"fixed_ip": schema.StringAttribute{
			MarkdownDescription: "The fixed (DHCP-reserved) IP assigned to the client, if any.",
			Computed:            true,
		},
		"network_id": schema.StringAttribute{
			MarkdownDescription: "The configured network ID for the client (set when `fixed_ip` is in use). " +
				"Reservations created through the controller UI carry no explicit `network_id` — the " +
				"DHCP scope is resolved from the address — so this falls back to " +
				"`last_connection_network_id`, keeping it consistent with `network_name`.",
			Computed: true,
		},
		"network_name": schema.StringAttribute{
			MarkdownDescription: "The name of the network the client was last seen on (controller field `last_connection_network_name`).",
			Computed:            true,
		},
		"is_wired": schema.BoolAttribute{
			MarkdownDescription: "Whether the client was last seen on a wired connection.",
			Computed:            true,
		},
		"is_guest": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is on a guest network.",
			Computed:            true,
		},
		"oui": schema.StringAttribute{
			MarkdownDescription: "The OUI (vendor) of the client's MAC address.",
			Computed:            true,
		},
		"blocked": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is blocked from network access.",
			Computed:            true,
		},
		"note": schema.StringAttribute{
			MarkdownDescription: "Free-text note attached to the client (the `note` field on `terrifi_client_device`).",
			Computed:            true,
		},
		"local_dns_record": schema.StringAttribute{
			MarkdownDescription: "The local DNS record hostname for the client, if one is configured.",
			Computed:            true,
		},
		"fixed_ap_mac": schema.StringAttribute{
			MarkdownDescription: "The MAC of the access point the client is locked to (if any).",
			Computed:            true,
		},
		"first_seen": schema.Int64Attribute{
			MarkdownDescription: "Unix timestamp when the controller first saw the client.",
			Computed:            true,
		},
		"last_seen": schema.Int64Attribute{
			MarkdownDescription: "Unix timestamp when the controller last saw the client.",
			Computed:            true,
		},
	}
}

func (d *clientsDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_clients"
}

func (d *clientsDataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the list of clients the UniFi controller knows about on a site — every record at `/api/s/{site}/rest/user`, which is the union of currently-connected stations and configured/remembered clients. Useful as a read-only liveness check on the controller and as a source for `for_each` loops over existing clients.",

		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				MarkdownDescription: "The site to read clients from. Defaults to the provider site.",
				Optional:            true,
				Computed:            true,
			},
			"clients": schema.ListNestedAttribute{
				MarkdownDescription: "The list of clients on the site. Ordering is controller-defined and not stable across reads.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: clientsEntrySchemaAttributes(),
				},
			},
		},
	}
}

func (d *clientsDataSource) Configure(
	_ context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *clientsDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data clientsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := d.client.SiteOrDefault(data.Site)

	clients, err := d.client.ListClientDevices(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Clients",
			fmt.Sprintf("Could not list clients in site %q: %s", site, err.Error()),
		)
		return
	}

	objects := make([]basetypes.ObjectValue, 0, len(clients))
	attrTypes := clientsEntryAttrTypes()
	for i := range clients {
		obj, diags := types.ObjectValue(attrTypes, clientsEntryValues(&clients[i]))
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objects = append(objects, obj)
	}

	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, objects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Site = types.StringValue(site)
	data.Clients = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// clientsEntryValues converts one unifi.Client into the attr.Value map the
// nested-object schema expects. Empty strings decode as null so users can
// distinguish "set" from "absent" — matches the device_data_source convention.
func clientsEntryValues(c *unifi.Client) map[string]attr.Value {
	blocked := false
	if c.Blocked != nil {
		blocked = *c.Blocked
	}

	// network_name already comes from the last-connection field; reading
	// network_id straight off the record made the pair disagree, reporting a
	// populated name next to a null id for every UI-created reservation (those
	// carry no explicit network_id — the controller resolves the DHCP scope
	// from the address). Use the same source for both.
	networkID := c.NetworkID
	if networkID == "" {
		networkID = c.LastConnectionNetworkID
	}

	return map[string]attr.Value{
		"id":               types.StringValue(c.ID),
		"mac":              types.StringValue(c.MAC),
		"name":             stringValueOrNull(c.Name),
		"hostname":         stringValueOrNull(c.Hostname),
		"ip":               stringValueOrNull(c.LastIP),
		"fixed_ip":         stringValueOrNull(c.FixedIP),
		"network_id":       stringValueOrNull(networkID),
		"network_name":     stringValueOrNull(c.LastConnectionNetworkName),
		"is_wired":         types.BoolValue(c.IsWired),
		"is_guest":         types.BoolValue(c.IsGuest),
		"oui":              stringValueOrNull(c.OUI),
		"blocked":          types.BoolValue(blocked),
		"note":             stringValueOrNull(c.Note),
		"local_dns_record": stringValueOrNull(c.LocalDNSRecord),
		"fixed_ap_mac":     stringValueOrNull(c.FixedApMAC),
		"first_seen":       types.Int64Value(c.FirstSeen),
		"last_seen":        types.Int64Value(c.LastSeen),
	}
}
