package com.aabiji.drip;

import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.net.Uri;
import android.os.Environment;
import android.provider.MediaStore;
import android.util.Log;

import java.io.OutputStream;

public class android_utility {
    public static String getDownloadsFolderPath() {
        return Environment.DIRECTORY_DOWNLOADS + "/";
    }

    public static void writeToPath(Context context, byte[] contents,
            String basePath, String filename, String mimetype) {
        try {
            ContentResolver resolver = context.getContentResolver();
            ContentValues values = new ContentValues();
            values.put(MediaStore.MediaColumns.DISPLAY_NAME, filename);
            values.put(MediaStore.MediaColumns.MIME_TYPE, mimetype);
            values.put(MediaStore.MediaColumns.RELATIVE_PATH,
                    basePath.endsWith("/") ? basePath : basePath + "/");

            Uri uri = resolver.insert(MediaStore.Files.getContentUri("external"), values);
            OutputStream output = resolver.openOutputStream(uri);
            output.write(contents);
            output.flush();
        } catch (Exception e) {
            Log.e("drip-debug", "Exception occurred!", e);
        }
    }
}
