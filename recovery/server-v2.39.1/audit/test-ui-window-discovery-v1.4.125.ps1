param([string]$Source)
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
$ast=[System.Management.Automation.Language.Parser]::ParseFile($Source,[ref]$tokens,[ref]$errors)
if($errors.Count -gt 0){throw ($errors.Message -join ', ')}
$function=$ast.Find({param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Select-ReceiptNativeWindow'},$true)
if(!$function){throw 'missing selector'}
. ([scriptblock]::Create($function.Extent.Text))
function Window([long]$handle,[long]$parent,[int]$owner,[bool]$visible,[string]$title){[pscustomobject]@{Handle=$handle;Parent=$parent;Process=$owner;Visible=$visible;Title=$title}}
$main=Window 1 0 42 $true 'PrimeERP'
$child=Window 2 1 42 $true '입고등록 - PG03F2120'
$wrongProcess=Window 3 0 99 $true '입고등록 - PG03F2120'
$hidden=Window 4 1 42 $false '입고등록 - PG03F2120'
if(@(Select-ReceiptNativeWindow @($main) 42).Count -ne 0){throw 'old top-level fixture must fail to find nested popup'}
$found=@(Select-ReceiptNativeWindow @($main,$child,$child,$wrongProcess,$hidden) 42)
if($found.Count -ne 1 -or $found[0].Handle -ne 2){throw 'nested popup selection or process/visibility/dedup failed'}
$other=Window 5 1 42 $true '입고등록 - PG03F2120'
if(@(Select-ReceiptNativeWindow @($child,$other) 42).Count -ne 2){throw 'ambiguous targets were silently merged'}
$sourceText=[IO.File]::ReadAllText($Source)
$native=[regex]::Match($sourceText,"(?s)Add-Type -TypeDefinition @'\r?\n(.*?)\r?\n'@")
if(!$native.Success){throw 'missing native enumeration source'}
# Compile P/Invoke declarations only. Windows APIs are never called on this host.
Add-Type -TypeDefinition $native.Groups[1].Value
'PASS: PowerShell AST and actual C# compile; synthetic nested HWND/process/hidden/dedup/ambiguity regression. Native Windows enumeration remains field verification.'
