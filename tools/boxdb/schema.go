package main

const schemaSQL = `
CREATE TABLE IF NOT EXISTS runtime_profile (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    core_name TEXT NOT NULL,
    mode TEXT NOT NULL,
    proxy_mode TEXT NOT NULL,
    auto_sync_config INTEGER NOT NULL,
    performance_mode INTEGER NOT NULL,
    clean_vendor_firewall INTEGER NOT NULL DEFAULT 0,
    ipv6_mode TEXT NOT NULL DEFAULT 'enable',
    config_name TEXT NOT NULL,
    quic TEXT NOT NULL,
    mihomo_dns_forward TEXT NOT NULL,
    proxy_tcp INTEGER NOT NULL,
    proxy_udp INTEGER NOT NULL,
    dns_hijack_tcp INTEGER NOT NULL,
    dns_hijack_udp INTEGER NOT NULL,
    dns_hijack_mode TEXT NOT NULL,
    cgroup_memcg INTEGER NOT NULL,
    memcg_limit TEXT NOT NULL,
    taskset_cpu INTEGER NOT NULL,
    allow_cpu TEXT NOT NULL,
    cgroup_blkio INTEGER NOT NULL,
    weight TEXT NOT NULL,
    bypass_cn INTEGER NOT NULL,
    boot_auto_start INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS app_selection (
    uid INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS app_gid_list (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cnip_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    bypass_cnip INTEGER NOT NULL,
    cnip_mode TEXT NOT NULL DEFAULT 'ipset',
    bypass_ipv4 INTEGER NOT NULL,
    bypass_ipv6 INTEGER NOT NULL,
    ipv4_file TEXT NOT NULL,
    ipv4_url TEXT NOT NULL,
    ipv6_file TEXT NOT NULL,
    ipv6_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cnip_force_uids (
    uid INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS cnip_force_apps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS network_match_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    network_control_enabled INTEGER NOT NULL,
    use_on_wifi_disconnect INTEGER NOT NULL,
    use_on_wifi_connect INTEGER NOT NULL,
    network_enable_on_unmatched INTEGER NOT NULL DEFAULT 1,
    network_enable_on_matched INTEGER NOT NULL DEFAULT 1,
    matched_action TEXT NOT NULL DEFAULT 'enable',
    unmatched_action TEXT NOT NULL DEFAULT 'stop',
    enable_ssid_matching INTEGER NOT NULL,
    enable_network_control_log INTEGER NOT NULL,
    list_mode TEXT NOT NULL,
    switch_config INTEGER NOT NULL DEFAULT 0,
    matched_config_name TEXT NOT NULL DEFAULT '',
    unmatched_config_name TEXT NOT NULL DEFAULT '',
    matched_script_name TEXT NOT NULL DEFAULT '',
    unmatched_script_name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS network_match_ssids (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS network_match_bssids (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS network_match_conditions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS network_rule_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    expression TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS network_rule_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    type TEXT NOT NULL,
    target TEXT NOT NULL DEFAULT '',
    continue_on_error INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_network_rule_actions_group ON network_rule_actions(group_id, sort_order);

CREATE TABLE IF NOT EXISTS shared_network_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    mac_filter INTEGER NOT NULL,
    mac_mode TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS shared_network_interface_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    initialized INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS shared_network_interfaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS blocked_interfaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS shared_network_macs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS intranet_bypass_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    initialized INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS intranet_ipv4_cidrs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS intranet_ipv6_cidrs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT NOT NULL
);
`

var defaultSharedInterfaces = []string{
	"ap+",
	"wlan+",
	"rndis+",
	"swlan+",
	"ncm+",
	"eth+",
}

var defaultIntranetV4 = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.0/24",
	"255.255.255.255/32",
}

var defaultIntranetV6 = []string{
	"::/128",
	"::1/128",
	"::ffff:0:0/96",
	"100::/64",
	"64:ff9b::/96",
	"2001::/32",
	"2001:10::/28",
	"2001:20::/28",
	"2001:db8::/32",
	"2002::/16",
	"fe80::/10",
	"ff00::/8",
}
