param([string]$Reader)
$ErrorActionPreference='Stop'
# Execute the production traversal with only native COM/PInvoke calls replaced
# by deterministic in-memory adapters. No real ERP data or Windows calls.
$s=[IO.File]::ReadAllText($Reader)
$s=$s.Replace('GetWindowThreadProcessId(new IntPtr(windowHandle), out owner);','FakeNative.Pid(new IntPtr(windowHandle), out owner);')
$s=$s.Replace('int result = AccessibleObjectFromWindow(','int result = FakeNative.Root(')
$s=$s.Replace('int childrenResult = AccessibleChildren(','int childrenResult = FakeNative.Children(')
$s=$s.Replace('Marshal.GetIUnknownForObject(accessible)','FakeNative.Identity(accessible)').Replace('Marshal.Release(identity)','FakeNative.Release(identity)')
$s+=@'
namespace Accessibility {
 public interface IAccessible {
  object get_accState(object id); object get_accRole(object id);
  string get_accName(object id); string get_accValue(object id);
  string get_accDefaultAction(object id); object get_accChild(object id);
  int accChildCount { get; }
 }
}
public class SampleNode : Accessibility.IAccessible {
 public uint Role; public uint State; public string Text=""; public int Id;
 public System.Collections.Generic.List<object> Children=new System.Collections.Generic.List<object>();
 public object get_accState(object id){return State;}
 public object get_accRole(object id){return Role;}
 public string get_accName(object id){if((State&0x20000000u)!=0)throw new Exception("PROTECTED TEXT READ");return Text;}
 public string get_accValue(object id){return get_accName(id);}
 public string get_accDefaultAction(object id){return Role==44?"Check":"";}
 public object get_accChild(object id){return null;}
 public int accChildCount {get{return Children.Count;}}
}
public static class FakeNative {
 public static SampleNode Tree;
 public static void Pid(IntPtr hwnd,out uint pid){pid=37;}
 public static int Root(IntPtr hwnd,uint oid,ref Guid iid,out object root){root=Tree;return 0;}
 public static int Children(Accessibility.IAccessible node,int start,int count,object[] children,out int obtained){
  SampleNode n=(SampleNode)node;obtained=Math.Min(count,n.Children.Count-start);
  for(int i=0;i<obtained;i++)children[i]=n.Children[start+i];return 0;
 }
 public static IntPtr Identity(object node){return new IntPtr(((SampleNode)node).Id);}
 public static void Release(IntPtr ignored){}
}
public static class SamplingRegression {
 static int nextId;
 static SampleNode Node(uint role,string text){return new SampleNode{Role=role,Text=text,Id=++nextId};}
 public static string Run(){
  var root=Node(10,"synthetic popup");var grid=Node(24,"synthetic grid");
  var secret=Node(42,"SECRET-MUST-NOT-READ");secret.State=0x20000000u;
  root.Children.Add(secret);root.Children.Add(grid);
  for(int row=0;row<1000;row++){
   var r=Node(28,"identical displayed purchase row");grid.Children.Add(r);
   for(int col=0;col<25;col++){
    var c=Node(col==0?44u:29u,col==0?"synthetic selection":"synthetic cell");
    if(col==0 && row==0)c.State=16;r.Children.Add(c);
   }
  }
  FakeNative.Tree=root;var report=AlphaReceiptMsaa.Read(1,37);
  int checks=0,cells=0;bool checkedSeen=false;var ordinals=new System.Collections.Generic.HashSet<int>();
  foreach(var e in report.Elements){
   if(e.Name.Contains("SECRET"))throw new Exception("secret exposed");
   if(e.Role==28)ordinals.Add(e.Ordinal);
   if(e.Role==44){checks++;if(e.DefaultAction!="Check")throw new Exception("capability lost");if(e.State==16)checkedSeen=true;}
   if(e.Role==29)cells++;
   if(e.Parent>=e.Index)throw new Exception("parent linkage invalid");
  }
  if(report.SampledRows!=8 || checks!=8 || cells!=192 || !checkedSeen || ordinals.Count!=8)throw new Exception("row cell sampling failed");
  if(!report.Truncated || report.SkippedRows<1 || report.Elements.Length>800 || report.PasswordExcluded!=1 || report.ReadErrors!=0)throw new Exception("bounds/protection failed");
  return "PASS: production traversal1000 duplicate-looking rows;8 complete row samples,8 checkboxes,192 cells, ordinals, checked state, optional action description, protected exclusion and partial-result flags. Native COM calls replaced only; no real UI action verified.";
 }
}
'@
Add-Type -TypeDefinition $s
[SamplingRegression]::Run()
