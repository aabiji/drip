package org.aabiji.drip;

import android.content.Context;
import android.content.ContentResolver;
import android.content.ContentValues;
import android.net.Uri;
import android.os.Environment;
import android.provider.MediaStore;

import java.io.OutputStream;

public class android_utility {
    public static void writeToDownloadsFolder(
        Context context, String filename, String mimetype, byte[] fileContents) {

        ContentResolver resolver = context.getContentResolver();

        ContentValues values = new ContentValues();
        values.put(MediaStore.Downloads.DISPLAY_NAME, filename);
        values.put(MediaStore.Downloads.MIME_TYPE, mimetype);
        values.put(MediaStore.Downloads.RELATIVE_PATH, Environment.DIRECTORY_DOWNLOADS);

        Uri uri = resolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values);
        if (uri != null) {
            try (OutputStream output = resolver.openOutputStream(uri)) {
                if (output != null) {
                    output.write(fileContents);
                    output.flush();
                }
            } catch (Exception e) {
                e.printStackTrace();
            }
        }
    }
}