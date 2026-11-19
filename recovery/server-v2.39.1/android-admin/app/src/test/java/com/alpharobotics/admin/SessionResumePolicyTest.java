package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class SessionResumePolicyTest {
 @Test public void savedAccountSelectsRestoreInsteadOfLogin(){assertTrue(SessionResumePolicy.saved(false,"grant","secret"));assertFalse(SessionResumePolicy.saved(true,"grant","secret"));assertFalse(SessionResumePolicy.saved(false,"","secret"));assertFalse(SessionResumePolicy.saved(false,"grant",""));}
 @Test public void restartedProcessStillNeedsServerProof(){UnlockPolicy state=new UnlockPolicy();assertTrue(SessionResumePolicy.start(true,state.isOpen(),false,true,false,"grant","secret"));assertFalse(state.isOpen());state.authenticated();assertFalse(SessionResumePolicy.start(true,state.isOpen(),false,true,false,"grant","secret"));}
 @Test public void logoutAndManualLoginCannotBeOverriddenByAutoRestore(){assertFalse(SessionResumePolicy.start(true,false,false,true,true,"grant","secret"));assertFalse(SessionResumePolicy.start(true,false,false,false,false,"grant","secret"));}
 @Test public void backgroundAndInflightRestoreCannotStartDuplicates(){assertFalse(SessionResumePolicy.start(false,false,false,true,false,"grant","secret"));assertFalse(SessionResumePolicy.start(true,false,true,true,false,"grant","secret"));assertTrue(SessionResumePolicy.start(true,false,false,true,false,"grant","secret"));}
}
