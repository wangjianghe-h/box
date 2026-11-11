#!/system/bin/sh

scripts_dir="${0%/*}"
settings_file="/data/adb/box/settings.ini"
moddir="/data/adb/modules/box_for_root"

[ -f "${settings_file}" ] && . "${settings_file}"

box_dir="/data/adb/box"
box_run="${box_dir}/run"
box_pid="${box_run}/box.pid"

wait_for_data_ready() {
  while [ ! -f "/data/system/packages.list" ] || [ ! -f "/data/system/packages.xml" ]; do
    sleep 1
  done
}

refresh_box() {
  if [ -f "${box_pid}" ]; then
    "${scripts_dir}/box.service" stop >/dev/null 2>&1
  fi
}

start_service() {
  [ -f "${settings_file}" ] && . "${settings_file}"

  if [ "${boot_auto_start}" = "false" ]; then
    log Info "开机自启已禁用，跳过启动核心服务。"
    return 0
  fi

  if [ -f "${moddir}/disable" ]; then
    log Info "模块已禁用，跳过启动核心服务。"
    return 0
  fi

  "${scripts_dir}/box.service" start
}

mkdir -p "${box_run}" "${box_run}/state" >/dev/null 2>&1 || true

if [ -f "${box_dir}/manual" ]; then
  rm -f "${box_pid}" >/dev/null 2>&1 || true
  exit 1
fi

if [ -f "${settings_file}" ] && [ -r "${settings_file}" ] && [ -s "${settings_file}" ]; then
  wait_for_data_ready
  refresh_box
  start_service
fi
