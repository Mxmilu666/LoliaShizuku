package com.wails.app;

import android.app.Activity;
import android.app.WallpaperColors;
import android.app.WallpaperManager;
import android.graphics.Color;
import android.os.Build;
import android.util.Log;
import android.view.View;
import android.view.Window;
import android.view.WindowInsetsController;
import org.json.JSONObject;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import android.webkit.JavascriptInterface;
import android.webkit.WebView;
import com.wails.app.BuildConfig;

/**
 * WailsJSBridge provides the JavaScript interface that allows the web frontend
 * to communicate with the Go backend. This is exposed to JavaScript as the
 * `window.wails` object.
 *
 * Similar to iOS's WKScriptMessageHandler but using Android's addJavascriptInterface.
 */
public class WailsJSBridge {
    private static final String TAG = "WailsJSBridge";
    private static final boolean DEBUG = BuildConfig.DEBUG;
    // Pooled threads avoid unbounded thread creation under high call volume.
    private static final ExecutorService executor = Executors.newCachedThreadPool();

    private final WailsBridge bridge;
    private final WebView webView;

    public WailsJSBridge(WailsBridge bridge, WebView webView) {
        this.bridge = bridge;
        this.webView = webView;
    }

    /**
     * Send a message to Go and return the response synchronously.
     * Called from JavaScript: wails.invoke(message)
     *
     * @param message The message to send (JSON string)
     * @return The response from Go (JSON string)
     */
    @JavascriptInterface
    public String invoke(String message) {
        if (DEBUG) Log.d(TAG, "Invoke called: " + message);
        return bridge.handleMessage(message);
    }

    /**
     * Send a message to Go asynchronously.
     * The response will be sent back via a callback.
     * Called from JavaScript: wails.invokeAsync(callbackId, message)
     *
     * @param callbackId The callback ID to use for the response
     * @param message The message to send (JSON string)
     */
    @JavascriptInterface
    public void invokeAsync(final String callbackId, final String payload) {
        if (DEBUG) Log.d(TAG, "InvokeAsync called: " + payload);

        // Handle off the JS thread so we don't block the WebView.
        executor.execute(() -> {
            try {
                String response = bridge.handleRuntimeCall(payload);
                sendCallback(callbackId, response, null);
            } catch (Exception e) {
                Log.e(TAG, "Error in async invoke", e);
                sendCallback(callbackId, null, e.getMessage());
            }
        });
    }

    /**
     * Log a message from JavaScript to Android's logcat
     * Called from JavaScript: wails.log(level, message)
     *
     * @param level The log level (debug, info, warn, error)
     * @param message The message to log
     */
    @JavascriptInterface
    public void log(String level, String message) {
        switch (level.toLowerCase()) {
            case "debug":
                Log.d(TAG + "/JS", message);
                break;
            case "info":
                Log.i(TAG + "/JS", message);
                break;
            case "warn":
                Log.w(TAG + "/JS", message);
                break;
            case "error":
                Log.e(TAG + "/JS", message);
                break;
            default:
                Log.v(TAG + "/JS", message);
                break;
        }
    }

    /**
     * Get the platform name
     * Called from JavaScript: wails.platform()
     *
     * @return "android"
     */
    @JavascriptInterface
    public String platform() {
        return "android";
    }

    /**
     * Set system bar colors so they blend with the app's top app bar / bottom
     * navigation bar (MD3). Called from JavaScript:
     * wails.setSystemBars('{"statusBar":"#rrggbb","navigationBar":"#rrggbb","dark":true}')
     * where "dark" means the app is using a dark theme (so bar icons are light).
     */
    @JavascriptInterface
    public void setSystemBars(final String json) {
        if (!(webView.getContext() instanceof Activity)) {
            return;
        }
        final Activity activity = (Activity) webView.getContext();
        activity.runOnUiThread(() -> {
            try {
                JSONObject opts = new JSONObject(json);
                boolean dark = opts.optBoolean("dark", false);
                Window window = activity.getWindow();
                if (opts.has("statusBar")) {
                    window.setStatusBarColor(Color.parseColor(opts.getString("statusBar")));
                }
                if (opts.has("navigationBar")) {
                    window.setNavigationBarColor(Color.parseColor(opts.getString("navigationBar")));
                }
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                    WindowInsetsController c = window.getInsetsController();
                    if (c != null) {
                        int mask = WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS
                                | WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS;
                        c.setSystemBarsAppearance(dark ? 0 : mask, mask);
                    }
                } else {
                    View decor = window.getDecorView();
                    int vis = decor.getSystemUiVisibility();
                    int flags = View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR;
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        flags |= View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR;
                    }
                    decor.setSystemUiVisibility(dark ? (vis & ~flags) : (vis | flags));
                }
            } catch (Exception e) {
                Log.e(TAG, "setSystemBars failed", e);
            }
        });
    }

    /**
     * Return the seed color for Material You (Monet) dynamic color. Called from
     * JavaScript: wails.getDynamicSeed() -> '{"seed":"#rrggbb","source":"system"}'
     * or "" when unavailable.
     *
     * Android 12+: system_accent1_500 from the system dynamic palette. The MD3
     * TonalSpot scheme depends only on the seed's hue, so feeding this color back
     * reproduces the system palette (it also follows a user-picked "basic color").
     * Android 8.1–11: the primary color of the current wallpaper.
     */
    @JavascriptInterface
    public String getDynamicSeed() {
        try {
            String source;
            int color;
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                color = webView.getContext().getColor(android.R.color.system_accent1_500);
                source = "system";
            } else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
                WallpaperColors colors = WallpaperManager.getInstance(webView.getContext())
                        .getWallpaperColors(WallpaperManager.FLAG_SYSTEM);
                if (colors == null) {
                    return "";
                }
                color = colors.getPrimaryColor().toArgb();
                source = "wallpaper";
            } else {
                return "";
            }
            JSONObject result = new JSONObject();
            result.put("seed", String.format("#%06x", color & 0xFFFFFF));
            result.put("source", source);
            return result.toString();
        } catch (Exception e) {
            Log.e(TAG, "getDynamicSeed failed", e);
            return "";
        }
    }

    /**
     * Check if we're running in debug mode
     * Called from JavaScript: wails.isDebug()
     *
     * @return true if debug build, false otherwise
     */
    @JavascriptInterface
    public boolean isDebug() {
        return BuildConfig.DEBUG;
    }

    /**
     * Send a callback response to JavaScript
     */
    private void sendCallback(String callbackId, String result, String error) {
        final String js;
        if (error != null) {
            js = String.format(
                    "window._wailsAndroidCallback && window._wailsAndroidCallback('%s', null, '%s');",
                    escapeJsString(callbackId),
                    escapeJsString(error)
            );
        } else {
            js = String.format(
                    "window._wailsAndroidCallback && window._wailsAndroidCallback('%s', '%s', null);",
                    escapeJsString(callbackId),
                    escapeJsString(result != null ? result : "")
            );
        }

        webView.post(() -> webView.evaluateJavascript(js, null));
    }

    private String escapeJsString(String str) {
        if (str == null) return "";
        return str.replace("\\", "\\\\")
                .replace("'", "\\'")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                // JS line terminators (U+2028/U+2029) must be escaped too; built via
                // (char) casts so the Java lexer does not reinterpret them as newlines.
                .replace(String.valueOf((char) 0x2028), "\\u2028")
                .replace(String.valueOf((char) 0x2029), "\\u2029");
    }
}
