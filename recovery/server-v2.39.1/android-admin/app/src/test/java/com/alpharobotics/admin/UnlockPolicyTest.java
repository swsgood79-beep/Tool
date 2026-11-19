package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class UnlockPolicyTest {
 @Test public void savedLoginDoesNotUnlockANewProcess(){UnlockPolicy state=new UnlockPolicy();assertFalse(LoginPolicy.mayPrompt(false,true,state.isOpen(),false));assertFalse(state.isOpen());}
 @Test public void explicitLogoutRequiresAuthenticationAgain(){UnlockPolicy state=new UnlockPolicy();state.authenticated();assertTrue(state.isOpen());state.lock();assertFalse(state.isOpen());state.authenticated();assertTrue(state.isOpen());}
 @Test public void cancellationAndNotificationDoNotOpenTheLockedApp(){UnlockPolicy state=new UnlockPolicy();state.authenticated();state.lock();assertFalse(state.isOpen());assertFalse(LoginPolicy.mayPrompt(false,true,state.isOpen(),false));assertFalse(state.isOpen());}
 @Test public void firstPasswordLoginAndExplicitOptOutDoNotRequirePriorFingerprint(){assertFalse(LoginPolicy.mayPrompt(true,false,false,false));UnlockPolicy state=new UnlockPolicy();state.authenticated();assertTrue(state.isOpen());}
}
