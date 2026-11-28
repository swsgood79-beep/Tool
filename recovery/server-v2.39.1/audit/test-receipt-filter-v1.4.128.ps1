param([string]$Reader,[string]$Script,[string]$AccessibilityReference,[string]$ReferenceDirectory,[string]$OutputAssembly)
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
[void][Management.Automation.Language.Parser]::ParseFile($Script,[ref]$tokens,[ref]$errors)
if($errors.Count){throw ($errors.Message -join ',')}
$source=[IO.File]::ReadAllText($Reader)
$references=@(Get-ChildItem (Join-Path $ReferenceDirectory '*.dll')|ForEach-Object{$_.FullName})+@($AccessibilityReference)
Add-Type -TypeDefinition $source -ReferencedAssemblies $references -OutputAssembly $OutputAssembly -OutputType Library
# Runtime selector uses same production source. Stub only Accessibility interface
# so Linux can load it; never execute the native filter or COM action methods.
$runtime=$source+@'
namespace Accessibility {
 public interface IAccessible {
  object get_accState(object id);object get_accRole(object id);
  string get_accName(object id);object get_accChild(object id);
  void accDoDefaultAction(object id);int accChildCount {get;}
 }
}
public static class FilterRegression {
 static AlphaReceiptFilter.Control C(long h,long parent,string name,string cls,int x,int y){return new AlphaReceiptFilter.Control{Handle=h,Parent=parent,Name=name,Class=cls,Bounds=new AlphaReceiptFilter.Box{Left=x,Top=y,Right=x+80,Bottom=y+25}};}
 public static string Run(){
  var items=new System.Collections.Generic.List<AlphaReceiptFilter.Control>();
  items.Add(C(1,100,"발주번호","STATIC",0,100));items.Add(C(2,100,"","EDIT",90,100));
  items.Add(C(3,100,"","EDIT",90,200));items.Add(C(4,200,"","EDIT",90,100));
  if(AlphaReceiptFilter.FindPurchaseEdit(items)!=2)throw new Exception("label match failed");
  items.Add(C(5,100,"","EDIT",180,100));if(AlphaReceiptFilter.FindPurchaseEdit(items)!=0)throw new Exception("ambiguous input accepted");items.RemoveAt(items.Count-1);
  items.Add(C(5,100,"발주번호","STATIC",0,300));if(AlphaReceiptFilter.FindPurchaseEdit(items)!=0)throw new Exception("duplicate label accepted");items.RemoveAt(items.Count-1);
  items.Insert(1,C(6,100,"다른 항목","STATIC",81,100));if(AlphaReceiptFilter.FindPurchaseEdit(items)!=0)throw new Exception("crossed unrelated label");
  return "PASS: actual labelled-control selector, duplicate label/input rejection, wrong parent/row exclusion, intervening-label rejection; official Accessibility compile and helper PowerShell AST. No native Windows calls/UI actions executed.";
 }
}
'@
Add-Type -TypeDefinition $runtime
[FilterRegression]::Run()
