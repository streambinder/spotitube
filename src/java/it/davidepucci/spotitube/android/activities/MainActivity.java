package it.davidepucci.spotitube.android.activities;

import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.support.design.widget.Snackbar;
import android.support.design.widget.TabLayout;
import android.support.v4.app.Fragment;
import android.support.v4.content.ContextCompat;
import android.support.v4.view.ViewPager;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.ArrayAdapter;
import android.widget.ListAdapter;
import android.widget.ListView;

import com.spotify.sdk.android.authentication.AuthenticationClient;
import com.spotify.sdk.android.authentication.AuthenticationRequest;
import com.spotify.sdk.android.authentication.AuthenticationResponse;

import org.json.JSONException;
import org.json.JSONObject;

import java.io.IOException;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.Locale;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import butterknife.BindView;
import butterknife.ButterKnife;
import it.davidepucci.spotitube.R;
import it.davidepucci.spotitube.android.callbacks.ReturningCallback;
import it.davidepucci.spotitube.android.pagers.PagerAdapter;
import it.davidepucci.spotitube.model.Track;
import okhttp3.Call;
import okhttp3.Callback;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;


public class MainActivity extends AppCompatActivity {

    // spotify
    public static final String CLIENT_ID = "d84f9faa18a84162ad6c73697990386c";
    public static final int AUTH_TOKEN_REQUEST_CODE = 0x10;
    private Locale locale;
    private String mAccessToken;
    private Call mCall;

    private final OkHttpClient mOkHttpClient = new OkHttpClient();
    private LinkedList<Track> library = new LinkedList<>();
    private PagerAdapter pagerAdapter;

    @BindView(R.id.activity_main)
    protected View relativeLayoutActivityMain;
    @BindView(R.id.toolbar)
    protected Toolbar toolbar;
    @BindView(R.id.tab_layout)
    protected TabLayout tabLayout;
    @BindView(R.id.pager)
    protected ViewPager viewPager;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        ButterKnife.bind(this);

        locale = getResources().getConfiguration().locale;

        setSupportActionBar(toolbar);

        tabLayout.addTab(tabLayout.newTab().setText("Libreria"));
        tabLayout.addTab(tabLayout.newTab().setText("Playlist"));
        tabLayout.setTabGravity(TabLayout.GRAVITY_FILL);

        pagerAdapter = new PagerAdapter(getSupportFragmentManager(), tabLayout.getTabCount());
        viewPager.setAdapter(pagerAdapter);
        viewPager.addOnPageChangeListener(new TabLayout.TabLayoutOnPageChangeListener(tabLayout));
        tabLayout.setOnTabSelectedListener(new TabLayout.OnTabSelectedListener() {
            @Override
            public void onTabSelected(TabLayout.Tab tab) {
                viewPager.setCurrentItem(tab.getPosition());
            }

            @Override
            public void onTabUnselected(TabLayout.Tab tab) {
            }

            @Override
            public void onTabReselected(TabLayout.Tab tab) {
            }
        });

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
                        final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Failed to fetch data: " + e, Snackbar.LENGTH_SHORT);
                        snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
                        snackbar.show();
                    }

                    @Override
                    public void onResponse(Call call, Response response) throws IOException {
                        try {
                            final JSONObject jsonObject = new JSONObject(response.body().string());
                            final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Connesso come: " + jsonObject.getString("display_name"), Snackbar.LENGTH_SHORT);
                            snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
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
                                    final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Failed to fetch data: " + e, Snackbar.LENGTH_SHORT);
                                    snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
                                    snackbar.show();
                                }

                                @Override
                                public void onResponse(Call call, Response response) throws IOException {
                                    try {
                                        final JSONObject jsonObject = new JSONObject(response.body().string());
                                        setResult(jsonObject.getInt("total"));
                                    } catch (JSONException e) {
                                        final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Failed to parse data: " + e, Snackbar.LENGTH_SHORT);
                                        snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
                                        snackbar.show();
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
                                        final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Failed to fetch data: " + e, Snackbar.LENGTH_SHORT);
                                        snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
                                        snackbar.show();
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
                                            }
                                        } catch (JSONException e) {
                                            final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Failed to parse data: " + e, Snackbar.LENGTH_SHORT);
                                            snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
                                            snackbar.show();
                                        }
                                        runOnUiThread(new Runnable() {
                                            @Override
                                            public void run() {
                                                synchronized (library) {
                                                    ArrayAdapter listAdapter = new ArrayAdapter<String>(getApplicationContext(), R.layout.list_row, library.stream()
                                                            .flatMap(t -> Stream.of(t.getTitle() + " - " + t.getArtist()))
                                                            .collect(Collectors.toList()));
                                                    final Fragment libraryFragment = pagerAdapter.getFragment(0);
                                                    if (libraryFragment.getView() != null) {
                                                        ListView listView = ButterKnife.findById(libraryFragment.getView(), R.id.mainListView);
                                                        listView.setAdapter(listAdapter);
                                                    }
                                                }
                                            }
                                        });
                                    }
                                });
                            }
                        } catch (JSONException e) {
                            final Snackbar snackbar = Snackbar.make(relativeLayoutActivityMain, "Failed to parse data: " + e, Snackbar.LENGTH_SHORT);
                            snackbar.getView().setBackgroundColor(ContextCompat.getColor(getApplicationContext(), R.color.colorSnackbar));
                            snackbar.show();
                        }
                    }
                });
            }
        }
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();

        return super.onOptionsItemSelected(item);
    }

    private AuthenticationRequest getAuthenticationRequest(AuthenticationResponse.Type type) {
        return new AuthenticationRequest.Builder(CLIENT_ID, type, getRedirectUri().toString())
                .setShowDialog(false)
                .setScopes(new String[]{"user-read-email", "user-library-read",
                        "playlist-read-private", "playlist-read-collaborative"})
                .build();
    }

    private void addToLibrary(final String text) {
        runOnUiThread(new Runnable() {
            @Override
            public void run() {
                final Fragment libraryFragment = pagerAdapter.getFragment(0);
                if (libraryFragment.getView() != null) {
                    ListView listView = ButterKnife.findById(libraryFragment.getView(), R.id.mainListView);
                }
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
