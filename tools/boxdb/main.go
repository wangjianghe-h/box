package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	home := flag.String("home", "/data/adb/box", "box home directory")
	dbPath := flag.String("db", "", "database path (default: <home>/box.db)")
	packagesFile := flag.String("packages-file", "/data/system/packages.list", "Android packages.list path")
	flag.Parse()

	if err := run(*home, *dbPath, *packagesFile); err != nil {
		fmt.Fprintln(os.Stderr, "boxdb:", err)
		os.Exit(1)
	}
}

func run(home, dbPath, packagesFile string) error {
	if home == "" {
		return fmt.Errorf("home directory is empty")
	}
	home = filepath.Clean(home)
	if dbPath == "" {
		dbPath = filepath.Join(home, "box.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	cfg, err := loadSettings(filepath.Join(home, "settings.ini"), home)
	if err != nil {
		return fmt.Errorf("read settings.ini: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("%s: %w", pragma, err)
		}
	}

	for _, statement := range strings.Split(schemaSQL, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create schema: %w", err)
		}
	}
	var schemaVersion int
	if err := db.QueryRow("PRAGMA user_version").Scan(&schemaVersion); err != nil {
		return fmt.Errorf("read database version: %w", err)
	}
	if schemaVersion < 13 {
		if _, err := db.Exec("PRAGMA user_version = 13"); err != nil {
			return fmt.Errorf("set database version: %w", err)
		}
	}

	packages := readPackages(packagesFile)
	selectedUIDs, cnipApps, cnipUIDs := resolveLists(home, cfg, packages)
	gids := readGidList(filepath.Join(home, "gid.list.cfg"))

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := syncRuntime(tx, cfg); err != nil {
		tx.Rollback()
		return err
	}
	if err := syncCnip(tx, cfg, home); err != nil {
		tx.Rollback()
		return err
	}
	if err := syncNetwork(tx, cfg); err != nil {
		tx.Rollback()
		return err
	}
	if err := syncShared(tx, cfg, home); err != nil {
		tx.Rollback()
		return err
	}
	if err := replaceInts(tx, "app_selection", "uid", selectedUIDs); err != nil {
		tx.Rollback()
		return err
	}
	if err := replaceInts(tx, "app_gid_list", "value", gids); err != nil {
		tx.Rollback()
		return err
	}
	if err := replaceInts(tx, "cnip_force_uids", "uid", cnipUIDs); err != nil {
		tx.Rollback()
		return err
	}
	if err := replaceStrings(tx, "cnip_force_apps", "value", cnipApps); err != nil {
		tx.Rollback()
		return err
	}
	if err := syncWifiLists(tx, cfg); err != nil {
		tx.Rollback()
		return err
	}
	if err := syncAppSettings(tx, cfg); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	_ = os.Chmod(dbPath, 0o644)
	fmt.Printf("boxdb: synced %s (uids=%d gids=%d cnip_uids=%d)\n", dbPath, len(selectedUIDs), len(gids), len(cnipUIDs))
	return nil
}

func syncRuntime(tx *sql.Tx, cfg *config) error {
	core := cfg.get("bin_name", "mihomo")
	mode := cfg.get("network_mode", "tun")
	proxyMode := cfg.get("proxy_mode", "core")
	configName := cfg.get("config_name", "")
	if configName == "" {
		configName = defaultConfigName(core, cfg)
	}
	ipv6Mode := normalizeIPV6Mode(cfg)
	quic := normalizeChoice(cfg.get("quic", "enable"), "enable", "enable", "disable")
	dnsForward := normalizeChoice(cfg.get("mihomo_dns_forward", "enable"), "enable", "enable", "disable")
	dnsMode := normalizeChoice(cfg.get("dns_hijack_mode", "tproxy"), "tproxy", "tproxy", "redirect", "disable")
	_, err := tx.Exec(`
INSERT INTO runtime_profile (
    id, core_name, mode, proxy_mode, auto_sync_config, performance_mode,
    clean_vendor_firewall, ipv6_mode, config_name, quic, mihomo_dns_forward,
    proxy_tcp, proxy_udp, dns_hijack_tcp, dns_hijack_udp, dns_hijack_mode,
    cgroup_memcg, memcg_limit, taskset_cpu, allow_cpu, cgroup_blkio, weight,
    bypass_cn, boot_auto_start
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    core_name = excluded.core_name,
    mode = excluded.mode,
    proxy_mode = excluded.proxy_mode,
    auto_sync_config = excluded.auto_sync_config,
    performance_mode = excluded.performance_mode,
    clean_vendor_firewall = excluded.clean_vendor_firewall,
    ipv6_mode = excluded.ipv6_mode,
    config_name = excluded.config_name,
    quic = excluded.quic,
    mihomo_dns_forward = excluded.mihomo_dns_forward,
    proxy_tcp = excluded.proxy_tcp,
    proxy_udp = excluded.proxy_udp,
    dns_hijack_tcp = excluded.dns_hijack_tcp,
    dns_hijack_udp = excluded.dns_hijack_udp,
    dns_hijack_mode = excluded.dns_hijack_mode,
    cgroup_memcg = excluded.cgroup_memcg,
    memcg_limit = excluded.memcg_limit,
    taskset_cpu = excluded.taskset_cpu,
    allow_cpu = excluded.allow_cpu,
    cgroup_blkio = excluded.cgroup_blkio,
    weight = excluded.weight,
    bypass_cn = excluded.bypass_cn,
    boot_auto_start = excluded.boot_auto_start`,
		core,
		mode,
		proxyMode,
		boolInt(cfg.boolean("auto_modify_config", true)),
		boolInt(cfg.boolean("performance_mode", false)),
		boolInt(cfg.boolean("clean_vendor_firewall", false)),
		ipv6Mode,
		configName,
		quic,
		dnsForward,
		boolInt(cfg.boolean("proxy_tcp", true)),
		boolInt(cfg.boolean("proxy_udp", true)),
		boolInt(cfg.boolean("dns_hijack_tcp", true)),
		boolInt(cfg.boolean("dns_hijack_udp", true)),
		dnsMode,
		boolInt(cfg.boolean("cgroup_memcg", false)),
		cfg.get("memcg_limit", "100M"),
		boolInt(cfg.boolean("taskset_cpu", false)),
		cfg.get("allow_cpu", "0-7"),
		boolInt(cfg.boolean("cgroup_blkio", false)),
		cfg.get("weight", ""),
		boolInt(cfg.boolean("bypass_cn_ip", false)),
		boolInt(cfg.boolean("boot_auto_start", false)),
	)
	if err != nil {
		return fmt.Errorf("sync runtime_profile: %w", err)
	}
	return nil
}

func syncCnip(tx *sql.Tx, cfg *config, home string) error {
	cnipMode := normalizeChoice(cfg.get("cnip_mode", "ipset"), "ipset", "ipset", "ebpf")
	_, err := tx.Exec(`
INSERT INTO cnip_settings (
    id, bypass_cnip, cnip_mode, bypass_ipv4, bypass_ipv6,
    ipv4_file, ipv4_url, ipv6_file, ipv6_url
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    bypass_cnip = excluded.bypass_cnip,
    cnip_mode = excluded.cnip_mode,
    bypass_ipv4 = excluded.bypass_ipv4,
    bypass_ipv6 = excluded.bypass_ipv6,
    ipv4_file = excluded.ipv4_file,
    ipv4_url = excluded.ipv4_url,
    ipv6_file = excluded.ipv6_file,
    ipv6_url = excluded.ipv6_url`,
		boolInt(cfg.boolean("bypass_cn_ip", false)),
		cnipMode,
		boolInt(cfg.boolean("bypass_cn_ip_v4", false)),
		boolInt(cfg.boolean("bypass_cn_ip_v6", false)),
		cfg.get("cn_ip_file", filepath.Join(home, "run", "state", "cn_ip.txt")),
		cfg.get("cn_ip_url", "https://github.com/Hackl0us/GeoIP2-CN/raw/release/CN-ip-cidr.txt"),
		cfg.get("cn_ipv6_file", filepath.Join(home, "run", "state", "cn_ipv6.txt")),
		cfg.get("cn_ipv6_url", "https://ispip.clang.cn/all_cn_ipv6.txt"),
	)
	if err != nil {
		return fmt.Errorf("sync cnip_settings: %w", err)
	}
	return nil
}

func syncNetwork(tx *sql.Tx, cfg *config) error {
	listMode := normalizeChoice(cfg.get("use_wifi_list_mode", "blacklist"), "blacklist", "blacklist", "whitelist")
	_, err := tx.Exec(`
INSERT INTO network_match_settings (
    id, network_control_enabled, use_on_wifi_disconnect, use_on_wifi_connect,
    network_enable_on_unmatched, network_enable_on_matched, matched_action, unmatched_action,
    enable_ssid_matching, enable_network_control_log, list_mode,
    switch_config, matched_config_name, unmatched_config_name,
    matched_script_name, unmatched_script_name
) VALUES (1, ?, ?, ?, 1, 1, 'enable', 'stop', ?, ?, ?, 0, '', '', '', '')
ON CONFLICT(id) DO UPDATE SET
    network_control_enabled = excluded.network_control_enabled,
    use_on_wifi_disconnect = excluded.use_on_wifi_disconnect,
    use_on_wifi_connect = excluded.use_on_wifi_connect,
    enable_ssid_matching = excluded.enable_ssid_matching,
    enable_network_control_log = excluded.enable_network_control_log,
    list_mode = excluded.list_mode`,
		boolInt(cfg.boolean("enable_network_service_control", false)),
		boolInt(cfg.boolean("use_module_on_wifi_disconnect", true)),
		boolInt(cfg.boolean("use_module_on_wifi", true)),
		boolInt(cfg.boolean("use_ssid_matching", false)),
		boolInt(cfg.boolean("inotify_log_enabled", true)),
		listMode,
	)
	if err != nil {
		return fmt.Errorf("sync network_match_settings: %w", err)
	}
	return nil
}

func syncShared(tx *sql.Tx, cfg *config, home string) error {
	macMode := normalizeChoice(cfg.get("mac_mode", "blacklist"), "blacklist", "blacklist", "whitelist")
	if _, err := tx.Exec(`
INSERT INTO shared_network_settings (id, mac_filter, mac_mode)
VALUES (1, ?, ?)
ON CONFLICT(id) DO UPDATE SET mac_filter = excluded.mac_filter, mac_mode = excluded.mac_mode`,
		boolInt(cfg.boolean("mac_filter", false)), macMode); err != nil {
		return fmt.Errorf("sync shared_network_settings: %w", err)
	}
	if _, err := tx.Exec(`
INSERT OR IGNORE INTO shared_network_interface_settings (id, initialized) VALUES (1, 1)`); err != nil {
		return fmt.Errorf("sync shared_network_interface_settings: %w", err)
	}
	if err := replaceStrings(tx, "shared_network_macs", "value", cfg.array("macs_list")); err != nil {
		return err
	}
	allowIfaces, ignoreIfaces := readAPList(filepath.Join(home, "ap.list.cfg"))
	var ifaceCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM shared_network_interfaces").Scan(&ifaceCount); err != nil {
		return err
	}
	if len(allowIfaces) == 0 && ifaceCount == 0 {
		allowIfaces = defaultSharedInterfaces
	}
	if len(allowIfaces) > 0 {
		if err := replaceStrings(tx, "shared_network_interfaces", "value", allowIfaces); err != nil {
			return err
		}
	}
	if err := replaceStrings(tx, "blocked_interfaces", "value", ignoreIfaces); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT OR IGNORE INTO intranet_bypass_settings (id, initialized) VALUES (1, 1)`); err != nil {
		return fmt.Errorf("sync intranet_bypass_settings: %w", err)
	}
	var v4Count, v6Count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM intranet_ipv4_cidrs").Scan(&v4Count); err != nil {
		return err
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM intranet_ipv6_cidrs").Scan(&v6Count); err != nil {
		return err
	}
	if v4Count == 0 {
		if err := replaceStrings(tx, "intranet_ipv4_cidrs", "value", defaultIntranetV4); err != nil {
			return err
		}
	}
	if v6Count == 0 {
		if err := replaceStrings(tx, "intranet_ipv6_cidrs", "value", defaultIntranetV6); err != nil {
			return err
		}
	}
	return nil
}

func syncWifiLists(tx *sql.Tx, cfg *config) error {
	ssids := cfg.array("wifi_ssids_list")
	bssids := cfg.array("wifi_bssids_list")
	if err := replaceStrings(tx, "network_match_ssids", "value", ssids); err != nil {
		return err
	}
	if err := replaceStrings(tx, "network_match_bssids", "value", bssids); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM network_match_conditions WHERE type IN ('wifi_ssid', 'wifi_bssid')"); err != nil {
		return fmt.Errorf("clear network_match_conditions: %w", err)
	}
	order := 0
	for _, ssid := range ssids {
		if _, err := tx.Exec("INSERT INTO network_match_conditions(type, value, enabled, sort_order) VALUES ('wifi_ssid', ?, 1, ?)", ssid, order); err != nil {
			return err
		}
		order++
	}
	for _, bssid := range bssids {
		if _, err := tx.Exec("INSERT INTO network_match_conditions(type, value, enabled, sort_order) VALUES ('wifi_bssid', ?, 1, ?)", bssid, order); err != nil {
			return err
		}
		order++
	}
	return nil
}

func syncAppSettings(tx *sql.Tx, cfg *config) error {
	defaults := [][2]string{
		{"app_language", cfg.get("app_language", "zh-CN")},
		{"clashapi_history_enabled", "false"},
		{"clashapi_history_retention_days", "30"},
		{"clashapi_history_max_database_mib", "128"},
	}
	for _, item := range defaults {
		if _, err := tx.Exec("INSERT OR IGNORE INTO app_settings(key, value) VALUES (?, ?)", item[0], item[1]); err != nil {
			return fmt.Errorf("sync app_settings: %w", err)
		}
	}
	return nil
}

func resolveLists(home string, cfg *config, packages map[string]int) ([]int, []string, []int) {
	selected := map[int]bool{}
	for _, entry := range readPackageList(filepath.Join(home, "package.list.cfg")) {
		if uid, _, ok := resolveEntry(entry, packages); ok {
			selected[uid] = true
		}
	}
	cnipApps := cfg.array("cnip_force_proxy_apps")
	cnipUIDs := map[int]bool{}
	appNames := make([]string, 0, len(cnipApps))
	for _, entry := range cnipApps {
		uid, name, ok := resolveEntry(entry, packages)
		if ok {
			cnipUIDs[uid] = true
		}
		if name != "" {
			appNames = append(appNames, name)
		} else if ok {
			appNames = append(appNames, entry)
		}
	}
	return sortedInts(selected), appNames, sortedInts(cnipUIDs)
}

func replaceInts(tx *sql.Tx, table, column string, values []int) error {
	if _, err := tx.Exec("DELETE FROM " + table); err != nil {
		return fmt.Errorf("clear %s: %w", table, err)
	}
	for _, value := range values {
		if _, err := tx.Exec("INSERT INTO "+table+"("+column+") VALUES (?)", value); err != nil {
			return fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return nil
}

func replaceStrings(tx *sql.Tx, table, column string, values []string) error {
	if _, err := tx.Exec("DELETE FROM " + table); err != nil {
		return fmt.Errorf("clear %s: %w", table, err)
	}
	for _, value := range values {
		if _, err := tx.Exec("INSERT INTO "+table+"("+column+") VALUES (?)", value); err != nil {
			return fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return nil
}

func sortedInts(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
