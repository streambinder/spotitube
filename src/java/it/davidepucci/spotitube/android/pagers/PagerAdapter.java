package it.davidepucci.spotitube.android.pagers;

import android.support.v4.app.Fragment;
import android.support.v4.app.FragmentManager;
import android.support.v4.app.FragmentStatePagerAdapter;

import java.util.ArrayList;

import it.davidepucci.spotitube.android.fragments.TabLibraryFragment;
import it.davidepucci.spotitube.android.fragments.TabPlaylistFragment;


public class PagerAdapter extends FragmentStatePagerAdapter {

    private int mNumOfTabs;
    private ArrayList<Fragment> fragments = new ArrayList<>();

    public PagerAdapter(FragmentManager fm, int NumOfTabs) {
        super(fm);
        this.mNumOfTabs = NumOfTabs;
    }

    public Fragment getFragment(int index) {
        if (index < fragments.size()) {
            return fragments.get(index);
        }
        return null;
    }

    @Override
    public Fragment getItem(int position) {
        switch (position) {
            case 0:
                TabLibraryFragment libraryFragment = new TabLibraryFragment();
                fragments.add(libraryFragment);
                return libraryFragment;
            case 1:
                TabPlaylistFragment playlistFragment = new TabPlaylistFragment();
                fragments.add(playlistFragment);
                return playlistFragment;
            default:
                return null;
        }
    }

    @Override
    public int getCount() {
        return mNumOfTabs;
    }
}