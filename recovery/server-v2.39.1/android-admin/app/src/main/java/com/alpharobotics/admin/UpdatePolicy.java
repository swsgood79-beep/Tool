package com.alpharobotics.admin;
import java.util.Arrays;
final class UpdatePolicy {
 static final String PREFIX="https://raw.githubusercontent.com/swsgood79-beep/Tool/recovery-v2.39.1/recovery/server-builds/";
 static boolean allowed(String url,String name,int code,String digest){return code>0&&name!=null&&name.matches("[0-9]+\\.[0-9]+\\.[0-9]+")&&url!=null&&url.equals(PREFIX+"AlphaAdmin-Android-v"+name+".apk")&&digest!=null&&digest.matches("[a-f0-9]{64}");}
 static boolean newer(int offered,long installed){return offered>installed;}
 static boolean sameSigners(byte[][] a,byte[][] b){if(a==null||b==null||a.length==0||a.length!=b.length)return false;boolean[] used=new boolean[b.length];for(byte[] x:a){boolean found=false;for(int i=0;i<b.length;i++)if(!used[i]&&Arrays.equals(x,b[i])){used[i]=true;found=true;break;}if(!found)return false;}return true;}
}
