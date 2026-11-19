package com.alpharobotics.admin;
import java.util.Locale;
import java.util.regex.*;
final class LoginPolicy {
 static boolean mayPrompt(boolean buttonPressed,boolean savedLogin,boolean open,boolean busy){return buttonPressed&&savedLogin&&!open&&!busy;}
 static String cookies(java.util.List<String> values){StringBuilder b=new StringBuilder();for(String value:values){int i=value.indexOf(';');String pair=i<0?value:value.substring(0,i);if(b.length()>0)b.append("; ");b.append(pair);}return b.toString();}
 static String username(String text){return text==null?"":text.trim().toLowerCase(Locale.ROOT);}
 static boolean fingerprintMatches(String entered,String saved){String e=username(entered),s=username(saved);return e.isEmpty()||!s.isEmpty()&&e.equals(s);}
 static String nonce(String html){Matcher m=Pattern.compile("name=\"csrf\"\\s+value=\"([A-Za-z0-9_-]{43})\"").matcher(html);return m.find()?m.group(1):"";}
 static boolean sameAccount(String expected,String actual){return expected!=null&&expected.equals(actual);}
}
