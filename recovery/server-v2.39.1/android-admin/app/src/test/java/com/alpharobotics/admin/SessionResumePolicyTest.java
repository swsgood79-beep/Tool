package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class SessionResumePolicyTest {
 @Test public void restartedProcessRestoresOnlyWithTheSavedCapabilityAndStillNeedsServerProof(){UnlockPolicy state=new UnlockPolicy();assertTrue(SessionResumePolicy.shouldRestore(true,state.isOpen(),false,false,"grant","secret"));assertFalse(state.isOpen());state.authenticated();assertFalse(SessionResumePolicy.shouldRestore(true,state.isOpen(),false,false,"grant","secret"));}
 @Test public void explicitLogoutOrMissingCredentialsCannotRestore(){assertFalse(SessionResumePolicy.shouldRestore(true,false,false,true,"grant","secret"));assertFalse(SessionResumePolicy.shouldRestore(true,false,false,false,"","secret"));assertFalse(SessionResumePolicy.shouldRestore(true,false,false,false,"grant",""));}
 @Test public void backgroundAndInflightRequestsCannotStartDuplicateRestores(){assertFalse(SessionResumePolicy.shouldRestore(false,false,false,false,"grant","secret"));assertFalse(SessionResumePolicy.shouldRestore(true,false,true,false,"grant","secret"));assertTrue(SessionResumePolicy.shouldRestore(true,false,false,false,"grant","secret"));}
}
