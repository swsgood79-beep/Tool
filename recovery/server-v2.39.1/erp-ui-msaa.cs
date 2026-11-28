using System;
using System.Collections.Generic;
using System.Globalization;
using System.Runtime.InteropServices;
using Accessibility;

// Read-only MSAA inspection of the already identified ERP receipt window.
// No selection, default action, focus, mouse or keyboard APIs are declared.
public static class AlphaReceiptMsaa
{
    public sealed class Element
    {
        public int Index;
        public int Parent;
        public int Depth;
        public int ChildId;
        public uint Role;
        public uint State;
        public string Name = "";
        public string Value = "";
        public int Children;
    }

    public sealed class Report
    {
        public string Reason = "unavailable";
        public Element[] Elements = new Element[0];
        public bool Truncated;
        public int ReadErrors;
        public int PasswordExcluded;
        public int ErrorCode;
    }

    private sealed class Pending
    {
        public IAccessible Accessible;
        public int Child;
        public int Parent;
        public int Depth;
    }

    [DllImport("oleacc.dll")]
    private static extern int AccessibleObjectFromWindow(
        IntPtr window, uint objectId, ref Guid interfaceId,
        [MarshalAs(UnmanagedType.Interface)] out object accessible);

    [DllImport("oleacc.dll")]
    private static extern int AccessibleChildren(
        [MarshalAs(UnmanagedType.Interface)] IAccessible container,
        int start, int count,
        [Out, MarshalAs(UnmanagedType.LPArray, SizeParamIndex = 2)] object[] children,
        out int obtained);

    private static string ReadText(Func<string> getter, Report report)
    {
        try
        {
            string text = getter() ?? "";
            if (text.Length > 1500)
            {
                report.Truncated = true;
                return text.Substring(0, 1500);
            }
            return text;
        }
        catch
        {
            report.ReadErrors++;
            return "";
        }
    }

    [DllImport("user32.dll")]
    private static extern uint GetWindowThreadProcessId(IntPtr window, out uint process);

    public static Report Read(long windowHandle, int expectedProcess)
    {
        Report report = new Report();
        if (windowHandle == 0)
        {
            report.Reason = "no_native_handle";
            return report;
        }

        uint owner;
        GetWindowThreadProcessId(new IntPtr(windowHandle), out owner);
        if (owner != (uint)expectedProcess)
        {
            report.Reason = "window_changed";
            return report;
        }

        Guid accessibleId = new Guid("618736E0-3C3D-11CF-810C-00AA00389B71");
        object rootObject;
        int result = AccessibleObjectFromWindow(
            new IntPtr(windowHandle), unchecked((uint)-4), ref accessibleId, out rootObject);
        if (result < 0 || !(rootObject is IAccessible))
        {
            report.ErrorCode = result;
            return report;
        }

        Queue<Pending> pending = new Queue<Pending>();
        pending.Enqueue(new Pending {
            Accessible = (IAccessible)rootObject, Child = 0, Parent = -1, Depth = 0
        });
        List<Element> elements = new List<Element>();
        HashSet<string> visited = new HashSet<string>();

        while (pending.Count > 0 && elements.Count < 800)
        {
            Pending entry = pending.Dequeue();
            IAccessible accessible = entry.Accessible;
            IntPtr identity = IntPtr.Zero;
            string key;
            try
            {
                identity = Marshal.GetIUnknownForObject(accessible);
                key = identity.ToInt64().ToString(CultureInfo.InvariantCulture) + ":" + entry.Child;
            }
            catch
            {
                report.ReadErrors++;
                continue;
            }
            finally
            {
                if (identity != IntPtr.Zero) Marshal.Release(identity);
            }
            if (!visited.Add(key)) continue;

            uint state;
            try
            {
                state = Convert.ToUInt32(accessible.get_accState(entry.Child), CultureInfo.InvariantCulture);
            }
            catch
            {
                // State must be readable before text: otherwise a protected
                // control cannot safely be distinguished from an ordinary one.
                report.ReadErrors++;
                continue;
            }
            if ((state & 0x20000000u) != 0)
            {
                report.PasswordExcluded++;
                continue;
            }

            Element element = new Element {
                Index = elements.Count, Parent = entry.Parent, Depth = entry.Depth,
                ChildId = entry.Child, State = state
            };
            try
            {
                element.Role = Convert.ToUInt32(accessible.get_accRole(entry.Child), CultureInfo.InvariantCulture);
            }
            catch { report.ReadErrors++; }
            element.Name = ReadText(delegate { return accessible.get_accName(entry.Child); }, report);
            element.Value = ReadText(delegate { return accessible.get_accValue(entry.Child); }, report);
            elements.Add(element);

            if (entry.Child != 0) continue;
            int childCount;
            try { childCount = accessible.accChildCount; }
            catch { report.ReadErrors++; continue; }
            element.Children = childCount;
            if (childCount <= 0) continue;
            if (entry.Depth >= 16 || pending.Count >= 1600)
            {
                report.Truncated = true;
                continue;
            }

            int count = Math.Min(childCount, Math.Min(800, 1600 - pending.Count));
            if (count < childCount) report.Truncated = true;
            object[] children = new object[count];
            int obtained;
            int childrenResult = AccessibleChildren(accessible, 0, count, children, out obtained);
            if (childrenResult < 0)
            {
                report.ReadErrors++;
                continue;
            }
            for (int index = 0; index < Math.Min(obtained, children.Length); index++)
            {
                IAccessible child = children[index] as IAccessible;
                if (child != null)
                {
                    pending.Enqueue(new Pending {
                        Accessible = child, Child = 0,
                        Parent = element.Index, Depth = entry.Depth + 1
                    });
                }
                else if (children[index] is int && (int)children[index] > 0)
                {
                    pending.Enqueue(new Pending {
                        Accessible = accessible, Child = (int)children[index],
                        Parent = element.Index, Depth = entry.Depth + 1
                    });
                }
            }
        }
        if (pending.Count > 0) report.Truncated = true;
        report.Elements = elements.ToArray();
        report.Reason = elements.Count > 0 ? "captured" : "unreadable";
        return report;
    }
}
