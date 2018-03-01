package it.davidepucci.spotitube.utils;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Map;

import it.davidepucci.spotitube.model.Track;
import it.davidepucci.spotitube.model.TrackType;

public class TrackUtils {

    public static HashSet<String> trackTypeAliases(TrackType trackType) {
        switch (trackType) {
            case Album:
                return new HashSet<>();
            case Live:
                return new HashSet<>(Arrays.asList(new String[]{"@", "live", "perform", "tour"}));
            case Cover:
                return new HashSet<>(Arrays.asList(new String[]{"cover", "vs"}));
            case Remix:
                return new HashSet<>(Arrays.asList(new String[]{"remix", "radio edit"}));
            case Acoustic:
                return new HashSet<>(Arrays.asList(new String[]{"acoustic"}));
            case Karaoke:
                return new HashSet<>(Arrays.asList(new String[]{"karaoke", "instrumental"}));
            case Parody:
                return new HashSet<>(Arrays.asList(new String[]{"parody"}));
        }
        return new HashSet<>();
    }

    public static boolean trackSeemsType(Track track, TrackType trackType) {
        for (String alias : trackTypeAliases(trackType)) {
            if (track.getTitle().toLowerCase().contains(alias)) {
                return true;
            }
        }
        return false;
    }
}
