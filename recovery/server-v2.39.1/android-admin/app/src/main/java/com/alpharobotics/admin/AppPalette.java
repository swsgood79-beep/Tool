package com.alpharobotics.admin;
final class AppPalette {
 static String normalize(String value){return "light".equals(value)||"dark".equals(value)?value:"system";}
 static boolean dark(String value,boolean systemDark){String v=normalize(value);return v.equals("dark")||v.equals("system")&&systemDark;}
 final int bg,surface,input,ink,muted,link,line,action;
 AppPalette(boolean dark){bg=dark?0xff10141c:0xfff3f5f8;surface=dark?0xff1b2330:0xffffffff;input=dark?0xff242e3e:0xffe9eef5;ink=dark?0xfff1f5fc:0xff172030;muted=dark?0xffb1bfd2:0xff5e6e83;link=dark?0xff9dc3ff:0xff1764cf;line=dark?0xff576980:0xffc1cbda;action=dark?0xff1e62c6:0xff1764cf;}
}
