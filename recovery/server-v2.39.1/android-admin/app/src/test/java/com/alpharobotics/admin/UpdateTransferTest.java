package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
import java.io.*;
import java.net.*;
import java.nio.file.Files;
import java.security.MessageDigest;
import java.util.*;

public class UpdateTransferTest {
 static final byte[] CONTENT="a verified update file".getBytes(java.nio.charset.StandardCharsets.UTF_8);
 static String hash(byte[] content)throws Exception{StringBuilder b=new StringBuilder();for(byte x:MessageDigest.getInstance("SHA-256").digest(content))b.append(String.format(Locale.ROOT,"%02x",x&255));return b.toString();}
 static class Connection extends HttpURLConnection {
  final int status;final byte[] bytes;final long size;boolean disconnected;boolean failRead;
  Connection(int status,byte[] bytes,long size)throws Exception{super(new URL("https://raw.githubusercontent.com/test"));this.status=status;this.bytes=bytes;this.size=size;}
  public void connect(){}public void disconnect(){disconnected=true;}public boolean usingProxy(){return false;}public int getResponseCode(){return status;}public long getContentLengthLong(){return size;}
  public InputStream getInputStream()throws IOException{if(failRead)throw new SocketTimeoutException("transient read timeout");return new ByteArrayInputStream(bytes);}
 }
 static File temp()throws IOException{return Files.createTempFile("alpha-update-unit-",".apk").toFile();}
 static UpdateTransfer.Progress quiet(){return(a,b,c)->{};}
 @Test public void startsImmediatelyAndReportsVerifiedProgress()throws Exception{File f=temp();try{Connection c=new Connection(200,CONTENT,CONTENT.length);List<Long> progress=new ArrayList<>();UpdateTransfer.download("url",hash(CONTENT),f,a->c,m->{fail("no backoff on success");},(a,b,t)->progress.add(b));assertArrayEquals(CONTENT,Files.readAllBytes(f.toPath()));assertEquals(Long.valueOf(0),progress.get(0));assertEquals(Long.valueOf(CONTENT.length),progress.get(progress.size()-1));assertTrue(c.disconnected);assertFalse(c.getInstanceFollowRedirects());}finally{f.delete();}}
 @Test public void transientServerFailureRetriesWithoutReopeningApp()throws Exception{File f=temp();try{int[] count={0};List<Long> pauses=new ArrayList<>();UpdateTransfer.download("url",hash(CONTENT),f,a->{try{return new Connection(++count[0]<3?503:200,CONTENT,CONTENT.length);}catch(Exception e){throw new IOException(e);}},pauses::add,quiet());assertEquals(3,count[0]);assertEquals(Arrays.asList(700L,1400L),pauses);assertArrayEquals(CONTENT,Files.readAllBytes(f.toPath()));}finally{f.delete();}}
 @Test public void timeoutRetriesAndDisconnectsFailedConnection()throws Exception{File f=temp();try{Connection first=new Connection(200,CONTENT,CONTENT.length);first.failRead=true;Connection good=new Connection(200,CONTENT,CONTENT.length);int[] n={0};UpdateTransfer.download("url",hash(CONTENT),f,a->n[0]++==0?first:good,m->{},quiet());assertEquals(2,n[0]);assertTrue(first.disconnected);assertTrue(good.disconnected);assertArrayEquals(CONTENT,Files.readAllBytes(f.toPath()));}finally{f.delete();}}
 @Test public void incompleteOrCorruptBodyNeverSurvivesAsReadyFile()throws Exception{File f=temp();try{int[] n={0};try{UpdateTransfer.download("url",hash(CONTENT),f,a->{n[0]++;try{return new Connection(200,new byte[]{1},CONTENT.length);}catch(Exception e){throw new IOException(e);}},m->{},quiet());fail("truncated update accepted");}catch(UpdateTransfer.Failure e){assertEquals("incomplete",e.getMessage());}assertEquals(3,n[0]);assertFalse(f.exists());}finally{f.delete();}}
 @Test public void digestMismatchRetriesButNeverAccepts()throws Exception{File f=temp();try{int[] n={0};try{UpdateTransfer.download("url",hash(CONTENT),f,a->{n[0]++;try{return new Connection(200,new byte[]{1},1);}catch(Exception e){throw new IOException(e);}},m->{},quiet());fail("unverified update accepted");}catch(UpdateTransfer.Failure e){assertEquals("integrity",e.getMessage());}assertEquals(3,n[0]);assertFalse(f.exists());}finally{f.delete();}}
 @Test public void notFoundDoesNotRetryOrLeavePartialFile()throws Exception{File f=temp();try{int[] n={0};try{UpdateTransfer.download("url",hash(CONTENT),f,a->{n[0]++;try{return new Connection(404,CONTENT,CONTENT.length);}catch(Exception e){throw new IOException(e);}},m->{fail("permanent error retried");},quiet());fail("404 accepted");}catch(UpdateTransfer.Failure e){assertEquals("http-404",e.getMessage());}assertEquals(1,n[0]);assertFalse(f.exists());}finally{f.delete();}}
 @Test public void oversizedDownloadIsRejectedBeforeReading()throws Exception{File f=temp();try{try{UpdateTransfer.download("url",hash(CONTENT),f,a->{try{return new Connection(200,CONTENT,UpdateTransfer.LIMIT+1);}catch(Exception e){throw new IOException(e);}},m->{fail("oversize retried");},quiet());fail("oversize accepted");}catch(UpdateTransfer.Failure e){assertEquals("too-large",e.getMessage());}assertFalse(f.exists());}finally{f.delete();}}
 @Test public void successfulRetryReplacesPartialBytes()throws Exception{File f=temp();try{int[] n={0};UpdateTransfer.download("url",hash(CONTENT),f,a->{try{return ++n[0]==1?new Connection(200,new byte[]{1},1):new Connection(200,CONTENT,-1);}catch(Exception e){throw new IOException(e);}},m->{},quiet());assertEquals(2,n[0]);assertArrayEquals(CONTENT,Files.readAllBytes(f.toPath()));}finally{f.delete();}}
}
