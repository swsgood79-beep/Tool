"""Package checked PC 1.4.128 source and Windows updater artifacts."""
from pathlib import Path
import hashlib
import json
import struct
import subprocess
import zipfile

root = Path(__file__).resolve().parent.parent
out = root.parent / 'server-builds'
version = '1.4.128'
stem = 'AlphaWorkComplete-Recovered'

def sha(data):
    return hashlib.sha256(data).hexdigest()

binary = Path('/workspace/work/AlphaWorkComplete-Recovered-v1.4.128.exe').read_bytes()
pe = struct.unpack_from('<I', binary, 0x3c)[0]
assert struct.unpack_from('<H', binary, pe + 24 + 68)[0] == 2
for name in ['go-full-v1.4.128.txt', 'go-race-v1.4.128.txt', 'browser-v1.4.128.txt']:
    log = (root / 'audit' / name).read_text()
    assert 'ok  ' in log and 'FAIL' not in log, name
for name in ['go-vet-linux-v1.4.128.txt', 'go-vet-windows-v1.4.128.txt']:
    assert not (root / 'audit' / name).read_text().strip(), name
browser = json.loads((root / 'audit/browser-release-v1.4.128.json').read_text())
assert len(browser['results']) == 4
assert all(result['result'] == 'pass' for result in browser['results'])
assert 'const releaseVersion = "1.4.128"' in (root / 'production.go').read_text()
apk = Path('/workspace/Tool/recovery/server-builds/AlphaAdmin-Android-v1.0.11.apk')
apk_hash = sha(apk.read_bytes())
assert apk_hash == 'afc14ea37edde0d3e9f78731f1ff60bc9eb75736a8f92aacc890b61ff73162ee'
assert json.loads((root / 'android-admin/update.json').read_text())['sha256'] == apk_hash
original = json.loads((root / 'audit/build-manifest-v0.2.json').read_text())['source_files']
assets = {name: checksum for name, checksum in original.items() if name.startswith('web/')}
assert len(assets) == 57
for name, checksum in assets.items():
    assert sha((root / name).read_bytes()) == checksum, name

manifest = {
    'pc_version': version,
    'apk_version': '1.0.11',
    'apk_changed': False,
    'apk_sha256': apk_hash,
    'source_commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
    'change': 'Admin outstanding selections/partial quantities, exact business rows/snapshots, grouped topmost Windows helper with vendor/PO copy and optional purchase-filter Search. Manual vendor/row/qty/final save. Native company flow pending.',
    'go_test_all': 'pass',
    'go_race_related': 'pass',
    'go_vet_linux': 'pass',
    'go_vet_windows': 'pass',
    'browser': browser,
    'original_assets_verified': 57,
    'windows_gui_exe_sha256': sha(binary),
    'windows_gui_exe_bytes': len(binary),
    'field_verification': 'Native company Windows helper launch/filter pending. Actual Alpha browser tested with synthetic rows and mocked native launcher. No automatic vendor/row/qty/final save.',
    'source_files': {str(path.relative_to(root)): sha(path.read_bytes()) for path in sorted(root.glob('*.go'))},
}
manifest_path = root / 'audit/build-manifest-v1.4.128.json'
manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n')

# The source archive is restricted to version-controlled files, plus the new
# generated verification manifest. No workspace data or runtime backups enter it.
tracked = subprocess.check_output(['git', 'ls-files', '-z', '--', '.'], cwd=root).decode().split('\0')
files = []
for name in tracked:
    if not name:
        continue
    path = root / name
    if not path.is_file() or path.name == 'SHA256SUMS':
        continue
    if path.suffix.lower() in {'.exe', '.dat', '.bak', '.bin', '.xls', '.xlsx', '.jks', '.keystore', '.apk', '.dpapi'}:
        continue
    if path.name in {'local.properties', 'google-services.json', 'keystore.properties'} or 'service-account' in path.name:
        continue
    if path.parent == root / 'audit' and path.suffix == '.png' and 'v1.4.128' not in path.name:
        continue
    files.append(path)
files = sorted(set(files + [manifest_path]))
checksums = root / 'SHA256SUMS'
checksums.write_text(''.join(f'{sha(path.read_bytes())}  {path.relative_to(root)}\n' for path in files))
name = f'{stem}-source-PC-v{version}-App-v1.0.11'
source = out / (name + '.zip')
windows = out / f'{stem}-Windows-x64-v{version}.zip'
for path in [source, windows]:
    assert not path.exists(), f'Existing releases must not be replaced: {path}'
with zipfile.ZipFile(source, 'w', zipfile.ZIP_DEFLATED, 9) as archive:
    for path in files + [checksums]:
        archive.write(path, name + '/' + str(path.relative_to(root)))
with zipfile.ZipFile(windows, 'w', zipfile.ZIP_DEFLATED, 9) as archive:
    archive.writestr(stem + '.exe', binary)
    for path in [root / 'README.md', root / 'START-HERE.md', root / 'CHANGELOG.md', root / 'PC-AUTO-UPDATE.md', root / 'audit/RECOVERY-V1.4.128.md', manifest_path]:
        archive.write(path, path.name)
    for path in sorted((root / 'docs/licenses').glob('*')):
        if path.is_file():
            archive.write(path, 'licenses/' + path.name)
    archive.writestr('SHA256SUMS', sha(binary) + '  ' + stem + '.exe\n')
for path in [source, windows]:
    with zipfile.ZipFile(path) as archive:
        assert archive.testzip() is None
    path.with_name(path.name + '.sha256').write_text(sha(path.read_bytes()) + '  ' + path.name + '\n')
    print(path.name, path.stat().st_size, sha(path.read_bytes()))

current = {
    'version': version,
    'url': 'https://raw.githubusercontent.com/swsgood79-beep/Tool/recovery-v2.39.1/recovery/server-builds/' + windows.name,
    'sha256': sha(windows.read_bytes()),
    'exe_sha256': sha(binary),
    'bytes': windows.stat().st_size,
    'notes': '관리자 미입고 품목 선택 · ERP 입력 도우미·발주 검색 지원 · APK1.0.11',
}
(out / 'pc-update.json').write_text(json.dumps(current, ensure_ascii=False, indent=2) + '\n')
