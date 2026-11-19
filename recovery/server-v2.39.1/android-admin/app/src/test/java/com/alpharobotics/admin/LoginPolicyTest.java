package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class LoginPolicyTest {
 @Test public void startingOrResumingNeverPromptsWithoutTheButton(){assertFalse(LoginPolicy.mayPrompt(false,true,false,false));assertTrue(LoginPolicy.mayPrompt(true,true,false,false));assertFalse(LoginPolicy.mayPrompt(true,false,false,false));assertFalse(LoginPolicy.mayPrompt(true,true,true,false));assertFalse(LoginPolicy.mayPrompt(true,true,false,true));}
 @Test public void aDifferentTypedUserCannotUseTheSavedFingerprint(){assertTrue(LoginPolicy.fingerprintMatches(" WSSONG ","wssong"));assertTrue(LoginPolicy.fingerprintMatches("","qa02"));assertFalse(LoginPolicy.fingerprintMatches("qa03","qa02"));assertFalse(LoginPolicy.fingerprintMatches("wssong",""));}
 @Test public void restoresOnlyTheBoundAccountIncludingTheOwnerPrincipal(){assertTrue(LoginPolicy.sameAccount("",""));assertTrue(LoginPolicy.sameAccount("id-qa02","id-qa02"));assertFalse(LoginPolicy.sameAccount("id-qa02",""));assertFalse(LoginPolicy.sameAccount("id-qa02","id-qa03"));}
 @Test public void proofUsesOnlyTheSuccessfulPasswordResponseCookie(){assertEquals("session=B",LoginPolicy.cookies(java.util.List.of("session=B; Path=/; Secure; HttpOnly")));assertEquals("session=B; nonce=N",LoginPolicy.cookies(java.util.List.of("session=B; Path=/","nonce=N; Path=/login")));}
 @Test public void normalLoginRequiresTheActualFormCsrfToken(){String token="A".repeat(43);assertEquals(token,LoginPolicy.nonce("<input type=\"hidden\" name=\"csrf\" value=\""+token+"\">"));assertEquals("",LoginPolicy.nonce("<input name=\"csrf\" value=\"short\">"));assertEquals("",LoginPolicy.nonce("<html>connection failed</html>"));}
}
