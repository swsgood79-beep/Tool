package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class UnlockPolicyTest {
 @Test public void savedLoginDoesNotUnlockANewProcess(){UnlockPolicy state=new UnlockPolicy();assertTrue(UnlockPolicy.needsUnlock(true,true));assertFalse(state.isOpen());}
 @Test public void successfulAuthenticationAllowsUseButBackgroundLocksAgain(){UnlockPolicy state=new UnlockPolicy();state.authenticated();assertTrue(state.isOpen());state.lock();assertFalse(state.isOpen());state.authenticated();assertTrue(state.isOpen());}
 @Test public void cancellationAndNotificationDoNotOpenTheLockedApp(){UnlockPolicy state=new UnlockPolicy();state.authenticated();state.lock();assertFalse(state.isOpen());assertTrue(UnlockPolicy.needsUnlock(true,true));assertFalse(state.isOpen());}
 @Test public void firstPasswordLoginAndExplicitOptOutDoNotRequirePriorFingerprint(){assertFalse(UnlockPolicy.needsUnlock(false,true));assertFalse(UnlockPolicy.needsUnlock(true,false));}
}
