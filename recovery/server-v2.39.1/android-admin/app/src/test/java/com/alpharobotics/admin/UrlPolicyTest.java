package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class UrlPolicyTest {
 @Test public void permitsOnlyExactHttpsTunnel(){assertEquals("https://company-test.trycloudflare.com",UrlPolicy.tunnel("https://company-test.trycloudflare.com"));for(String bad:new String[]{"http://company-test.trycloudflare.com","https://company-test.trycloudflare.com/","https://company-test.trycloudflare.com:443","https://company-test.trycloudflare.com@evil.test","https://company-test.trycloudflare.com.evil.test","javascript:alert(1)","https://company-test.trycloudflare.com?q=x"})assertNull(bad,UrlPolicy.tunnel(bad));}
 @Test public void rejectsCredentialAndHostInjection(){String base="https://company-test.trycloudflare.com";assertTrue(UrlPolicy.sameOrigin(base+"/login",base));assertFalse(UrlPolicy.sameOrigin("https://x@company-test.trycloudflare.com/",base));assertFalse(UrlPolicy.sameOrigin("https://evil.test/",base));assertFalse(UrlPolicy.sameOrigin("file:///etc/passwd",base));assertEquals("/mobile-admin",UrlPolicy.inspection(-1,4));assertEquals("/projects/2/inspection?request=7",UrlPolicy.inspection(2,7));}
}
