package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class UpdatePolicyTest {
 @Test public void onlyTheExactReleaseSourceAndDigestCanBeOffered(){String name="1.0.3",sha="a".repeat(64),url=UpdatePolicy.PREFIX+"AlphaAdmin-Android-v"+name+".apk";assertTrue(UpdatePolicy.allowed(url,name,4,sha));for(String bad:new String[]{"https://evil.test/app.apk",url+"?redirect=1",url.replace("https:","http:"),url.replace("1.0.3","1.0.4")})assertFalse(UpdatePolicy.allowed(bad,name,4,sha));assertFalse(UpdatePolicy.allowed(url,name,4,"x".repeat(64)));}
 @Test public void anUpdateMustActuallyIncreaseTheInstalledVersion(){assertTrue(UpdatePolicy.newer(4,3));assertFalse(UpdatePolicy.newer(3,3));assertFalse(UpdatePolicy.newer(2,3));}
 @Test public void alteredOrMissingApkSignaturesAreRejected(){byte[][] installed={{1,2,3},{4,5,6}};assertTrue(UpdatePolicy.sameSigners(installed,new byte[][]{{4,5,6},{1,2,3}}));assertFalse(UpdatePolicy.sameSigners(installed,new byte[][]{{1,2,3}}));assertFalse(UpdatePolicy.sameSigners(installed,new byte[][]{{1,2,3},{7,8,9}}));assertFalse(UpdatePolicy.sameSigners(installed,null));}
}
