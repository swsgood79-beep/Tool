package com.alpharobotics.admin;

import java.io.*;
import java.net.*;
import java.security.MessageDigest;
import java.util.Locale;

/** Bounded, retryable transfer. A file is never usable before its digest passes. */
final class UpdateTransfer {
 static final long LIMIT=32L*1024*1024;
 interface Connections { HttpURLConnection open(String address)throws IOException; }
 interface Pause { void waitMs(long millis)throws InterruptedException; }
 interface Progress { void update(int attempt,long bytes,long total); }
 static class Failure extends IOException {
  final boolean retryable;
  Failure(String message,boolean retryable){super(message);this.retryable=retryable;}
 }
 static void download(String address,String expected,File target,Connections connections,Pause pause,Progress progress)throws IOException,InterruptedException{
  IOException last=null;
  for(int attempt=1;attempt<=3;attempt++){
   progress.update(attempt,0,-1);
   try{once(address,expected,target,connections,progress,attempt);return;}
   catch(IOException e){last=e;if(target.exists()&&!target.delete())throw new Failure("temporary-file",false);
    if(e instanceof Failure&&!((Failure)e).retryable||attempt==3)throw e;
    pause.waitMs(attempt*700L);
   }
  }
  throw last==null?new IOException("download"):last;
 }
 private static void once(String address,String expected,File target,Connections connections,Progress progress,int attempt)throws IOException{
  HttpURLConnection c=connections.open(address);c.setInstanceFollowRedirects(false);c.setConnectTimeout(15000);c.setReadTimeout(30000);c.setRequestProperty("Cache-Control","no-cache");
  try{
   int status=c.getResponseCode();if(status!=200)throw new Failure("http-"+status,status==408||status==425||status==429||status>=500);
   long total=c.getContentLengthLong();if(total>LIMIT)throw new Failure("too-large",false);
   MessageDigest digest;try{digest=MessageDigest.getInstance("SHA-256");}catch(Exception e){throw new Failure("digest-unavailable",false);}
   long size=0,lastNotice=0;progress.update(attempt,0,total);
   try(InputStream in=c.getInputStream();OutputStream out=new FileOutputStream(target)){
    byte[] buffer=new byte[16384];int n;
    while((n=in.read(buffer))!=-1){size+=n;if(size>LIMIT)throw new Failure("too-large",false);out.write(buffer,0,n);digest.update(buffer,0,n);
     if(size-lastNotice>=65536){progress.update(attempt,size,total);lastNotice=size;}
    }
   }
   if(size==0||total>=0&&size!=total)throw new Failure("incomplete",true);
   StringBuilder hex=new StringBuilder();for(byte b:digest.digest())hex.append(String.format(Locale.ROOT,"%02x",b&255));
   if(!expected.equals(hex.toString()))throw new Failure("integrity",true);
   progress.update(attempt,size,total);
  }finally{c.disconnect();}
 }
}
