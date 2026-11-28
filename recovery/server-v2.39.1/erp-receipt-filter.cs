using System;
using System.Collections.Generic;
using System.Globalization;
using System.Runtime.InteropServices;
using System.Text;
using Accessibility;

// Only the labelled purchase filter and exact toolbar Search may be acted on.
// No row selection, quantity editing or final receipt save is performed.
public static class AlphaReceiptFilter {
 public struct Box {public int Left,Top,Right,Bottom;}
 public sealed class Control {public long Handle,Parent;public bool Protected;public string Name="",Class="";public Box Bounds;}
 private delegate bool EnumProc(IntPtr hwnd,IntPtr arg);
 [DllImport("user32.dll")] private static extern bool EnumWindows(EnumProc callback,IntPtr arg);
 [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr root,EnumProc callback,IntPtr arg);
 [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hwnd);
 [DllImport("user32.dll")] private static extern bool IsWindowEnabled(IntPtr hwnd);
 [DllImport("user32.dll")] private static extern bool IsChild(IntPtr parent,IntPtr child);
 [DllImport("user32.dll")] private static extern IntPtr GetParent(IntPtr hwnd);
 [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd,out uint process);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetWindowText(IntPtr hwnd,StringBuilder text,int capacity);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr hwnd,StringBuilder text,int capacity);
 [DllImport("user32.dll",EntryPoint="GetWindowLongW")] private static extern int GetWindowLong(IntPtr hwnd,int index);
 [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr hwnd,out Box box);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern IntPtr SendMessageTimeout(IntPtr hwnd,uint message,IntPtr wParam,string text,uint flags,uint timeout,out UIntPtr result);
 [DllImport("oleacc.dll")] private static extern int AccessibleObjectFromWindow(IntPtr window,uint objectId,ref Guid iid,[MarshalAs(UnmanagedType.Interface)] out object accessible);
 [DllImport("oleacc.dll")] private static extern int AccessibleChildren([MarshalAs(UnmanagedType.Interface)] IAccessible parent,int start,int count,[Out,MarshalAs(UnmanagedType.LPArray,ArraySubType=UnmanagedType.Struct,SizeParamIndex=2)] object[] children,out int obtained);
 private static string Text(IntPtr h){var s=new StringBuilder(512);GetWindowText(h,s,s.Capacity);return s.ToString();}
 private static string Class(IntPtr h){var s=new StringBuilder(256);GetClassName(h,s,s.Capacity);return s.ToString();}
 private static bool Owned(IntPtr h,int pid){uint p;GetWindowThreadProcessId(h,out p);return p==(uint)pid&&IsWindowVisible(h)&&IsWindowEnabled(h);}
 private static List<Control> Controls(IntPtr popup,int pid){
  var items=new List<Control>();bool exceeded=false;
  EnumChildWindows(popup,delegate(IntPtr h,IntPtr arg){if(items.Count>=800){exceeded=true;return false;}if(!Owned(h,pid))return true;
   string cls=Class(h);Box b;if(GetWindowRect(h,out b))items.Add(new Control{Handle=h.ToInt64(),Parent=GetParent(h).ToInt64(),Name=cls.ToUpperInvariant().Contains("STATIC")?Text(h):"",Class=cls,Protected=cls.ToUpperInvariant().Contains("EDIT")&&(GetWindowLong(h,-16)&0x20)!=0,Bounds=b});return true;
  },IntPtr.Zero);return exceeded?new List<Control>():items;
 }
 // Relative control bounds identify the labelled input; no coordinate clicks.
 public static long FindPurchaseEdit(List<Control> controls){
  var labels=controls.FindAll(delegate(Control c){return c.Name.Trim()=="발주번호";});if(labels.Count!=1)return 0;
  var label=labels[0];var edits=new List<Control>();
  foreach(var c in controls){
   if(c.Protected||c.Parent!=label.Parent||!c.Class.ToUpperInvariant().Contains("EDIT"))continue;
   long center=((long)c.Bounds.Top+c.Bounds.Bottom)/2;
   if(c.Bounds.Left<label.Bounds.Right||center<label.Bounds.Top||center>label.Bounds.Bottom)continue;
   bool intervening=false;
   foreach(var other in controls)if(other.Parent==label.Parent&&other.Name!=""&&other.Handle!=label.Handle&&other.Bounds.Left>=label.Bounds.Right&&other.Bounds.Left<c.Bounds.Left&&center>=other.Bounds.Top&&center<=other.Bounds.Bottom)intervening=true;
   if(!intervening)edits.Add(c);
  }
  return edits.Count==1?edits[0].Handle:0;
 }
 private sealed class Entry {public IAccessible Node;public int Child,Depth;public bool Toolbar;}
 private static Entry SearchButton(IntPtr popup){
  Guid iid=new Guid("618736E0-3C3D-11CF-810C-00AA00389B71");object root;
  if(AccessibleObjectFromWindow(popup,unchecked((uint)-4),ref iid,out root)<0||!(root is IAccessible))return null;
  var queue=new Queue<Entry>();queue.Enqueue(new Entry{Node=(IAccessible)root});var found=new List<Entry>();var visited=new HashSet<string>();int count=0;
  while(queue.Count>0&&count++<400){
   var e=queue.Dequeue();IntPtr identity=IntPtr.Zero;string key;
   try{identity=Marshal.GetIUnknownForObject(e.Node);key=identity.ToInt64()+":"+e.Child;}finally{if(identity!=IntPtr.Zero)Marshal.Release(identity);}
   if(!visited.Add(key))continue;
   try{
    uint state=Convert.ToUInt32(e.Node.get_accState(e.Child),CultureInfo.InvariantCulture);
    if((state&(0x20000000u|0x1u|0x8000u|0x10000u))!=0)continue;
    uint role=Convert.ToUInt32(e.Node.get_accRole(e.Child),CultureInfo.InvariantCulture);
    if(role==24||role==28||role==29)continue;
    // The known popup toolbar is shallow. Do not explore grid clients or
    // unrelated editors while looking for its Search command.
    if(!e.Toolbar&&role!=22&&e.Depth>=3)continue;
    string name=(e.Node.get_accName(e.Child)??"").Replace("&","").Trim();
    if(e.Toolbar&&role==43&&name=="검색")found.Add(e);
    if(e.Child!=0||e.Depth>=12)continue;
    int total=e.Node.accChildCount;if(total>100)return null;if(total<=0)continue;
    object[] children=new object[total];int obtained;
    if(AccessibleChildren(e.Node,0,total,children,out obtained)<0||obtained!=total)return null;
    for(int i=0;i<obtained;i++){
     IAccessible child=children[i] as IAccessible;int childId=0;
     if(child==null&&children[i] is int){childId=(int)children[i];if(childId<=0)continue;try{child=e.Node.get_accChild(childId) as IAccessible;}catch{}if(Object.ReferenceEquals(child,e.Node))child=null;if(child!=null)childId=0;}
     if(child==null&&childId==0)continue;
     queue.Enqueue(new Entry{Node=child??e.Node,Child=childId,Depth=e.Depth+1,Toolbar=e.Toolbar||role==22});
    }
   }catch{return null;}
  }
  return queue.Count==0&&found.Count==1?found[0]:null;
 }
 public static string Apply(int pid,string purchase){
  if(String.IsNullOrWhiteSpace(purchase)||purchase.Length>40)return "발주번호를 확인할 수 없습니다";
  foreach(char c in purchase)if(!Char.IsLetterOrDigit(c)&&c!='-'&&c!='_')return "발주번호 형식을 확인해 주세요";
  var matches=new HashSet<long>();
  EnumProc test=delegate(IntPtr h,IntPtr arg){if(Owned(h,pid)&&!Class(h).ToUpperInvariant().Contains("EDIT")&&!Class(h).ToUpperInvariant().Contains("TEXTBOX")&&Text(h).Trim().EndsWith("PG03F2120",StringComparison.Ordinal))matches.Add(h.ToInt64());return true;};
  EnumWindows(delegate(IntPtr h,IntPtr arg){uint owner;GetWindowThreadProcessId(h,out owner);if(owner==(uint)pid){test(h,arg);EnumChildWindows(h,test,arg);}return true;},IntPtr.Zero);
  if(matches.Count!=1)return "ERP Data 불러오기 창 한 개를 열어 주세요. 발주번호 복사로도 검색할 수 있습니다";
  long target=0;foreach(long h in matches)target=h;IntPtr popup=new IntPtr(target);
  long editHandle=FindPurchaseEdit(Controls(popup,pid));
  if(editHandle==0)return "발주번호 입력칸을 고유하게 찾지 못했습니다. 발주번호 복사로 직접 검색해 주세요";
  var search=SearchButton(popup);if(search==null)return "검색 버튼을 고유하게 확인하지 못했습니다. 발주번호 복사로 직접 검색해 주세요";
  IntPtr edit=new IntPtr(editHandle);
  if(!Owned(popup,pid)||!Owned(edit,pid)||!IsChild(popup,edit)||FindPurchaseEdit(Controls(popup,pid))!=editHandle)return "ERP 창이 변경됐습니다. 다시 확인해 주세요";
  UIntPtr result;
  if(SendMessageTimeout(edit,0x000C,IntPtr.Zero,purchase,0x0002,2000,out result)==IntPtr.Zero||Text(edit)!=purchase)return "발주번호 입력을 확인하지 못했습니다. ERP 입력칸을 확인해 주세요";
  if(!Owned(popup,pid)||(search.Node.get_accName(search.Child)??"").Replace("&","").Trim()!="검색")return "ERP 검색 대상을 다시 확인해 주세요";
  try{search.Node.accDoDefaultAction(search.Child);}catch{return "발주번호는 입력했습니다. ERP 검색 버튼을 직접 눌러 주세요";}
  return "발주 검색을 실행했습니다. 거래처·품번·발주 행·수량을 확인해 체크하고 ERP 저장을 직접 눌러 주세요";
 }
}
