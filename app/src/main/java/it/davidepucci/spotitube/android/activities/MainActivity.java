package it.davidepucci.spotitube.android.activities;

import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.support.design.widget.Snackbar;
import android.support.v4.content.ContextCompat;
import android.support.v7.app.AppCompatActivity;
import android.widget.TextView;

import com.spotify.sdk.android.authentication.AuthenticationClient;
import com.spotify.sdk.android.authentication.AuthenticationRequest;
import com.spotify.sdk.android.authentication.AuthenticationResponse;

import org.json.JSONException;
import org.json.JSONObject;

import java.io.IOException;
import java.util.LinkedList;
import java.util.Locale;

import it.davidepucci.spotitube.R;
import it.davidepucci.spotitube.android.callbacks.ReturningCallback;
import it.davidepucci.spotitube.model.Track;
import okhttp3.Call;
import okhttp3.Callback;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;


public class MainActivity extends AppCompatActivity {

    public static final String CLIENT_ID = "d84f9faa18a84162ad6c73697990386c";
    public static final int AUTH_TOKEN_REQUEST_CODE = 0x10;
    private final OkHttpClient mOkHttpClient = new OkHttpClient();

    private Locale locale;
    private String mAccessToken;
    private Call mCall;

    private LinkedList<Track> library = new LinkedList<>();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        locale = getResources().getConfiguration().locale;

        final AuthenticationRequest request = getAuthenticationRequest(AuthenticationResponse.Type.TOKEN);
        AuthenticationClient.openLoginActivity(this, AUTH_TOKEN_REQUEST_CODE, request);
    }

    @Override
    protected void onDestroy() {
        cancelCall();
        super.onDestroy();
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        final AuthenticationResponse response = AuthenticationClient.getResponse(resultCode, data);
        if (AUTH_TOKEN_REQUEST_CODE == requestCode) {
            mAccessToken = response.getAccessToken();
            if (mAccessToken != null) {
                final Request request = new Request.Builder()
                        .url("https://api.spotify.com/v1/me")
                        .addHeader("Authorization", "Bearer " + mAccessToken)
                        .build();

                cancelCall();
                mCall = mOkHttpClient.newCall(request);

                mCall.enqueue(new Callback() {
                    @Override
                    public void onFailure(Call call, IOException e) {
                        setResponse("Failed to fetch data: " + e);
                    }

                    @Override
                    public void onResponse(Call call, Response response) throws IOException {
                        try {
                            final JSONObject jsonObject = new JSONObject(response.body().string());
                            final Snackbar snackbar = Snackbar.make(findViewById(R.id.activity_main),
                                    jsonObject.getString("display_name")
                                            + " <" + jsonObject.getString("email") + ">", Snackbar.LENGTH_SHORT);
                            snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorAccent));
                            snackbar.show();

                            final Request requestLibraryTotals = new Request.Builder()
                                    .url("https://api.spotify.com/v1/me/tracks?offset=0&limit=1")
                                    .addHeader("Authorization", "Bearer " + mAccessToken)
                                    .build();

                            cancelCall();
                            mCall = mOkHttpClient.newCall(requestLibraryTotals);
                            ReturningCallback<Integer> mCallBack = new ReturningCallback<Integer>() {
                                @Override
                                public void onFailure(Call call, IOException e) {
                                    setResponse("Failed to fetch data: " + e);
                                }

                                @Override
                                public void onResponse(Call call, Response response) throws IOException {
                                    try {
                                        final JSONObject jsonObject = new JSONObject(response.body().string());
                                        setResult(jsonObject.getInt("total"));
                                    } catch (JSONException e) {
                                        setResponse("Failed to parse data: " + e);
                                    }
                                }
                            };
                            mCall.enqueue(mCallBack);

                            Integer totals = null;
                            while (totals == null) {
                                totals = mCallBack.getResult();
                            }
                            for (int offset = 0; offset < totals; offset += 50) {
                                final Request requestLibrary = new Request.Builder()
                                        .url("https://api.spotify.com/v1/me/tracks?offset=" + String.valueOf(offset)
                                                + "&limit=50&market=" + locale.getCountry())
                                        .addHeader("Authorization", "Bearer " + mAccessToken)
                                        .build();

                                //cancelCall();
                                mCall = mOkHttpClient.newCall(requestLibrary);

                                mCall.enqueue(new Callback() {
                                    @Override
                                    public void onFailure(Call call, IOException e) {
                                        setResponse("Failed to fetch data: " + e);
                                    }

                                    @Override
                                    public void onResponse(Call call, Response response) throws IOException {
                                        try {
                                            final JSONObject jsonObject = new JSONObject(response.body().string());
                                            for (int i = 0; i < jsonObject.getJSONArray("items").length(); i++) {
                                                Track track = new Track(jsonObject.getJSONArray("items").getJSONObject(i).getJSONObject("track"));
                                                synchronized (library) {
                                                    library.add(track);
                                                }
                                                setResponse(track.toString());
                                            }
                                        } catch (JSONException e) {
                                            setResponse("Failed to parse data: " + e);
                                        }
                                    }
                                });
                            }
                        } catch (JSONException e) {
                            setResponse("Failed to parse data: " + e);
                        }
                    }
                });
            }
        }
    }

    private AuthenticationRequest getAuthenticationRequest(AuthenticationResponse.Type type) {
        return new AuthenticationRequest.Builder(CLIENT_ID, type, getRedirectUri().toString())
                .setShowDialog(false)
                .setScopes(new String[]{"user-read-email", "user-library-read",
                        "playlist-read-private", "playlist-read-collaborative"})
                .build();
    }

    private void setResponse(final String text) {
        runOnUiThread(new Runnable() {
            @Override
            public void run() {
                final TextView responseView = (TextView) findViewById(R.id.response_text_view);
                responseView.append(text + "\n");
            }
        });
    }


    private void cancelCall() {
        if (mCall != null) {
            mCall.cancel();
        }
    }

    private Uri getRedirectUri() {
        return Uri.parse("http://localhost:8080/callback");
    }
}
