param([string]$Source,[string]$Reader,[string]$AccessibilityReference,[string]$ReferenceDirectory,[string]$OutputAssembly)
$ErrorActionPreference='Stop'
$tokens=$null;$parseErrors=$null
$ast=[System.Management.Automation.Language.Parser]::ParseFile($Source,[ref]$tokens,[ref]$parseErrors)
if($parseErrors.Count){throw ($parseErrors.Message -join ', ')}
$legacyBlocks=@($ast.FindAll({param($node) $node -is [System.Management.Automation.Language.TryStatementAst] -and $node.Extent.Text.Contains('if($legacyType -and')},$true) | Sort-Object {$_.Extent.Text.Length})
if(!$legacyBlocks.Count){throw 'missing managed compatibility guard'}
$block=[scriptblock]::Create($legacyBlocks[0].Extent.Text)
$legacyType=$null;$element=[pscustomobject]@{};$pattern=$null;$report=[ordered]@{read_errors=0}
for($index=0;$index -lt 24;$index++){. $block}
if($report.read_errors -ne 0){throw 'unsupported managed legacy type still creates one error per node'}
$readerText=[IO.File]::ReadAllText($Reader)
$references=@(Get-ChildItem (Join-Path $ReferenceDirectory '*.dll') | ForEach-Object {$_.FullName}) + @($AccessibilityReference)
Add-Type -TypeDefinition $readerText -ReferencedAssemblies $references -OutputAssembly $OutputAssembly -OutputType Library
'PASS: missing-managed-type guard on24 synthetic elements has zero false errors. MSAA C# compiled against official Accessibility contract. Windows native calls not executed.'
