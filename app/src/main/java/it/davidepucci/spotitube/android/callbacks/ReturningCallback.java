package it.davidepucci.spotitube.android.callbacks;

import java.io.IOException;

import okhttp3.Call;
import okhttp3.Callback;
import okhttp3.Response;

public abstract class ReturningCallback<T> implements Callback {

    private T result;

    public T getResult() {
        return result;
    }

    public void setResult(T object) {
        result = object;
    }

    @Override
    public abstract void onFailure(Call call, IOException e);

    @Override
    public abstract void onResponse(Call call, Response response) throws IOException;
}
