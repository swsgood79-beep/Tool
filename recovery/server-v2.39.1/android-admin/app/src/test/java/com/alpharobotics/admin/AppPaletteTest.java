package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class AppPaletteTest {
 @Test public void explicitLightOverridesDarkDevice(){assertFalse(AppPalette.dark("light",true));assertTrue(AppPalette.dark("dark",false));}
 @Test public void systemTracksDeviceAndUnknownValuesUseSystem(){assertTrue(AppPalette.dark("system",true));assertFalse(AppPalette.dark("system",false));assertTrue(AppPalette.dark(null,true));assertFalse(AppPalette.dark("invalid",false));}
 @Test public void darkNativeSurfacesAndTextAreDifferentPalettes(){AppPalette dark=new AppPalette(true),light=new AppPalette(false);assertNotEquals(dark.surface,light.surface);assertEquals(0xff1b2330,dark.surface);assertEquals(0xfff1f5fc,dark.ink);assertEquals(0xffffffff,light.surface);}
}
