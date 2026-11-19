package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class AuthEpochTest {
 @Test public void logoutRejectsPendingAccountAndPushResponses(){AuthEpoch e=new AuthEpoch();long bootstrap=e.current(),push=e.current();e.invalidate();assertFalse(e.accepts(bootstrap));assertFalse(e.accepts(push));}
 @Test public void newLoginCannotBeOverwrittenByThePreviousLogin(){AuthEpoch e=new AuthEpoch();long old=e.invalidate(),current=e.invalidate();assertFalse(e.accepts(old));assertTrue(e.accepts(current));e.invalidate();assertFalse(e.accepts(current));}
}
