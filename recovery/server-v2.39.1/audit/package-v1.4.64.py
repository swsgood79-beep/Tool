"""Package verified release without replacing earlier archives."""
from pathlib import Path
import hashlib,json,re,struct,zipfile
root=Path(__file__).resolve().parent.parent
out=root.parent/'server-builds';version='1.4.64';stem='AlphaWorkComplete-Recovered'
exe=Path('/workspace/recovery-input')/f'{stem}-v{version}.exe'
sha=lambda b:hashlib.sha256(b).hexdigest()
b=exe.read_bytes();pe=struct.unpack_from('<I',b,0x3c)[0];assert struct.unpack_from('<H',b,pe+24+68)[0]==2
assets={n:h for n,h in json.loads((root/'audit/build-manifest-v0.2.json').read_text())['source_files'].items() if n.startswith('web/')};assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
log=(root/f'audit/go-test-final-v{version}.txt').read_text();assert 'ok  ' in log and '\nFAIL' not in log
browser=json.loads((root/'audit/browser-worker-submissions-v1.4.61.json').read_text());assert len(browser['results'])==52 and all(x['result']=='pass' for x in browser['results'])
regression=json.loads((root/'audit/browser-worker-reconnect-v1.4.51.json').read_text());assert len(regression['results'])==22 and all(x['result']=='pass' for x in regression['results'])
actual=(root/'audit/worker-submissions-browser-v1.4.61.txt').read_text();assert '\nPASS\n' in actual and '\nFAIL' not in actual
shipping=json.loads((root/'audit/browser-shipping-records-v1.4.51.json').read_text());assert len(shipping['results'])==19 and all(x['result']=='pass' for x in shipping['results'])
assert '\nPASS\n' in (root/'audit/shipping-browser-v1.4.51.txt').read_text()
archive=json.loads((root/'audit/browser-archive-v1.4.51.json').read_text());assert len(archive['results'])==16 and all(x['result']=='pass' for x in archive['results'])
assert '\nPASS\n' in (root/'audit/archive-browser-v1.4.51.txt').read_text()
reset=json.loads((root/'audit/browser-archive-reset-v1.4.53.json').read_text());assert len(reset['results'])==12 and all(x['result']=='pass' for x in reset['results'])
assert '\nPASS\n' in (root/'audit/archive-reset-browser-v1.4.53.txt').read_text()
cancel=json.loads((root/'audit/browser-archive-cancel-v1.4.53.json').read_text());assert len(cancel['results'])==12 and all(x['result']=='pass' for x in cancel['results'])
assert '\nPASS\n' in (root/'audit/archive-cancel-browser-v1.4.53.txt').read_text()
accounts=json.loads((root/'audit/browser-admin-accounts-v1.4.55.json').read_text());assert len(accounts['results'])==13 and all(x['result']=='pass' for x in accounts['results'])
assert '\nPASS\n' in (root/'audit/admin-accounts-browser-v1.4.55.txt').read_text()
navigation=json.loads((root/'audit/browser-calendar-navigation-v1.4.57.json').read_text());assert len(navigation['results'])==28 and all(x['result']=='pass' for x in navigation['results'])
assert '\nPASS\n' in (root/'audit/calendar-navigation-browser-v1.4.57.txt').read_text()
alarms=json.loads((root/'audit/browser-worker-alarm-v1.4.58.json').read_text());assert len(alarms['results'])==48 and all(x['result']=='pass' for x in alarms['results'])
assert '\nPASS\n' in (root/'audit/worker-alarm-browser-v1.4.58.txt').read_text()
grid=json.loads((root/'audit/browser-admin-grid-v1.4.59.json').read_text());assert len(grid['results'])==25 and all(x['result']=='pass' for x in grid['results'])
assert '\nPASS\n' in (root/'audit/admin-grid-browser-v1.4.59.txt').read_text()
limit=json.loads((root/'audit/browser-shipping-limit-v1.4.60.json').read_text());assert len(limit['results'])==8 and all(x['result']=='pass' for x in limit['results'])
assert '\nPASS\n' in (root/'audit/shipping-limit-browser-v1.4.60.txt').read_text()
app=json.loads((root/'audit/browser-worker-app-v1.4.63.json').read_text());assert len(app['results'])==12 and all(x['result']=='pass' for x in app['results'])
sw=json.loads((root/'audit/browser-app-sw-v1.4.63.json').read_text());assert len(sw['results'])==3 and all(x['result']=='pass' for x in sw['results'])
assert '\nPASS\n' in (root/'audit/worker-app-browser-v1.4.63.txt').read_text()
assert '\nPASS\n' in (root/'audit/app-security-race-v1.4.61.txt').read_text()
viewer=json.loads((root/'audit/browser-shipment-viewer-v1.4.62.json').read_text());assert len(viewer['results'])==24 and all(x['result']=='pass' for x in viewer['results'])
assert '\nPASS\n' in (root/'audit/shipment-viewer-browser-v1.4.62.txt').read_text()
theme=json.loads((root/'audit/browser-app-theme-v1.4.63.json').read_text());assert len(theme['results'])==24 and all(x['result']=='pass' for x in theme['results'])
assert '\nPASS\n' in (root/'audit/app-theme-browser-v1.4.63.txt').read_text()
apk=out/'AlphaAdmin-Android-v1.0.0.apk'
assert 'BUILD SUCCESSFUL' in (root/'audit/android-release-build-v1.4.64.txt').read_text()
assert '0 errors, 2 warnings' in (root/'audit/android-lint-v1.4.64.txt').read_text()
assert 'PASS' in (root/'audit/native-admin-race-v1.4.64.txt').read_text()
assert 'Verified using v2 scheme (APK Signature Scheme v2): true' in (root/'audit/android-apk-signature-v1.4.64.txt').read_text()
manifest={'native_admin_android':{'version':'1.0.0','package':'com.alpharobotics.admin','min_sdk':26,'target_sdk':35,'apk_bytes':apk.stat().st_size,'apk_sha256':sha(apk.read_bytes()),'release_build':'pass','junit_tests':2,'lint_errors':0,'lint_warnings':2,'signature_v2':'pass','physical_phone_and_live_fcm':'not tested; user Firebase initial setup required','persistent_login':'server capabilities stored hashed; Android Keystore AES-GCM; manual logout or account/device revocation invalidates; normal browser session TTL unchanged'},'native_auth_push_and_race':'pass','unchanged_worker_pwa_reference_version':'1.4.63','installed_app_theme_browser':theme,'unchanged_shipment_viewer_reference_version':'1.4.62','shipment_photo_viewer_browser':viewer,'installed_app_current_version':'1.4.63','installed_worker_app_browser':app,'app_service_worker_browser':sw,'pages_app':json.loads((root/'audit/pages-worker-app-v1.4.63.json').read_text()),'unchanged_photo_limit_reference_version':'1.4.60','shipping_thirty_photo_browser':limit,'unchanged_admin_grid_reference_version':'1.4.59','admin_card_grid_browser':grid,'unchanged_worker_alarm_reference_version':'1.4.58','worker_all_page_alarm_browser':alarms,'unchanged_calendar_reference_version':'1.4.57','calendar_navigation_browser':navigation,'unchanged_accounts_reference_version':'1.4.55','unchanged_worker_submission_reference_version':'1.4.61','individual_account_browser':accounts,'unchanged_archive_cancel_reference_version':'1.4.53','archive_cancel_browser':cancel,'archive_reset_browser':reset,'unchanged_regression_reference_version':'1.4.51','archive_browser':archive,'worker_submission_browser':browser,'publication':'GitHub recovery-v2.39.1 source and Windows download package','shipping_browser':shipping,'version':version,'change':'Add administrator-only signed Android APK with data-only FCM, owner Firebase setup, persistent Keystore login grants, explicit logout/revocation; preserve existing browser sessions and worker app','exe_sha256':sha(b),'exe_bytes':len(b),'target':'windows/amd64 GUI CGO_ENABLED=0','original_assets_verified':57,'go_test_all':'pass','go_test_race_reference_version':'1.4.54','optional_tests_skipped':re.findall('^--- SKIP: (.+)$',log,re.M),'go_vet_linux':'pass','go_vet_windows':'pass','worker_reconnect_browser':regression,'pages_bridge':json.loads((root/'audit/pages-talk-reader-v1.4.47.json').read_text()),'windows_execution':'not tested; cross-build only','operating_data':'existing operating directory or explicitly scheduled verified custom directory; existing configured data directory respected; new optional ShippingRecords extension and independent photos; no historical completed-order migration','preview_preference':'browser-local alpha-design-default-v2; default modern; optional legacy fallback; theme follows device with manual override','native_window':'existing Win32 layout preserved; this release promotes the modern web UI','source_files':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted(root.glob('*.go'))},'extra_assets':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted((root/'extras').rglob('*')) if p.is_file()}}
(root/f'audit/build-manifest-v{version}.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.name not in {'local.properties','google-services.json','keystore.properties'} and not any(x in p.parts for x in {'.gradle','build'}) and p.suffix not in {'.exe','.dat','.bak','.bin','.xlsx','.jks','.keystore','.apk'} and 'service-account' not in p.name]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
source=out/f'{stem}-source-v{version}.zip';windows=out/f'{stem}-Windows-x64-v{version}.zip'
for p in [source,windows]:assert not p.exists(),f'Never overwrite {p}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,f'{stem}-source-v{version}/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr(stem+'.exe',b)
 for p in [root/'README.md',root/'START-HERE.md',root/'ANDROID-ADMIN-SETUP.md',root/f'audit/RECOVERY-V{version}.md',root/f'audit/build-manifest-v{version}.json']:z.write(p,p.name)
 z.writestr('SHA256SUMS',sha(b)+'  '+stem+'.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n');print(p.name,p.stat().st_size,sha(p.read_bytes()))
