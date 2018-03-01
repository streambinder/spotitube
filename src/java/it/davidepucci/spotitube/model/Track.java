package it.davidepucci.spotitube.model;

import org.json.JSONException;
import org.json.JSONObject;

import java.io.File;
import java.util.LinkedList;

import it.davidepucci.spotitube.utils.TrackUtils;

public class Track {

    private String title;
    private String song;
    private String artist;
    private String album;
    private String year;
    private LinkedList<String> featurings = new LinkedList<>();
    private String genre;
    private Integer trackNumber;
    private Integer trackTotals;
    private Integer duration;
    private TrackType trackType;
    private String image;
    private String url;
    private String filename;
    private String filenameTemp;
    private String filenameExt;
    private String searchPattern;
    private String lyrics;
    private Boolean local;

    public Track(JSONObject trackObject) {
        try {
            title = trackObject.getString("name");
            artist = trackObject.getJSONArray("artists").getJSONObject(0).getString("name");
            album = trackObject.getJSONObject("album").getString("name");
            year = trackObject.getJSONObject("album").getString("release_date").substring(0, 4);
            for (int i = 1; i < trackObject.getJSONArray("artists").length(); i++) {
                featurings.add(trackObject.getJSONArray("artists").getJSONObject(i).getString("name"));
            }
            /*for (int i = 0; i < trackObject.getJSONObject("album").getJSONArray("genres").length(); i++) {
                featurings.add(trackObject.getJSONObject("album").getJSONArray("genres").get(i).toString());
            }*/
            trackNumber = trackObject.getInt("track_number");
            //trackTotals = trackObject.getJSONObject("album").getJSONArray("tracks").length();
            duration = trackObject.getInt("duration_ms") / 1000;
            image = trackObject.getJSONObject("album").getJSONArray("images").getJSONObject(0).getString("url");
            filenameExt = "mp3";
            local = false;

            for (TrackType trackType : TrackType.values()) {
                if (TrackUtils.trackSeemsType(this, trackType)) {
                    this.trackType = trackType;
                }
            }

            for (String separator : new String[]{"-", "live"}) {
                title = title.split(" " + separator + " ")[0];
            }

            if (featurings.size() > 0) {
                if (title.toLowerCase().contains("feat. ") ||
                        title.toLowerCase().contains("ft. ") ||
                        title.toLowerCase().contains("featuring ") ||
                        title.toLowerCase().contains("with ")) {
                    for (String featSymbol : new String[]{"featuring", "feat.", "with"}) {
                        title = title.replaceAll(featSymbol + " ", "ft. ");
                    }
                } else {
                    if (title.toLowerCase().contains("(") && title.toLowerCase().contains(")") &&
                            (title.toLowerCase().contains(" vs. ") || title.toLowerCase().contains(" vs "))) {
                        title = title.split(" \\(")[0];
                    }
                    String inlineFeaturings = new String();
                    if (featurings.size() > 1) {
                        inlineFeaturings += String.join(", ", featurings.subList(0, featurings.size() - 2))
                                + " and " + featurings.get(featurings.size() - 1);
                    } else {
                        inlineFeaturings += featurings.get(0);
                    }
                    title = title + " (ft. " + inlineFeaturings + ")";
                }
                song = title.split(" \\(ft. ")[0];
            } else {
                song = title;
            }

            /*album = album.replaceAll("\\[", "(");
            album = album.replaceAll("]", ")");
            album = album.replaceAll("\\{", "(");
            album = album.replaceAll("}", "(");

            filename = artist + " - " + title;
            for (String symbol : new String[]{"/", "\\", ".", "?", "<", ">", ":", "*"}) {
                filename = filename.replaceAll(symbol, "");
            }
            filename = filename.replaceAll("\\ \\ ", " ");*/
            filenameTemp = "." + filename;

            searchPattern = filenameTemp.replaceAll("\\-", " ");

            File file = new File(filenameFinal());
            if (file.exists() && !file.isDirectory()) {
                local = true;
            }
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    private String filenameFinal() {
        return filename + filenameExt;
    }

    @Override
    public String toString() {
        return "\"" + title + "\" in \"" + album + "\" by \"" + artist + "\" (" + year + ")";
    }

    public String getTitle() {
        return title;
    }

    public void setTitle(String title) {
        this.title = title;
    }

    public String getSong() {
        return song;
    }

    public void setSong(String song) {
        this.song = song;
    }

    public String getArtist() {
        return artist;
    }

    public void setArtist(String artist) {
        this.artist = artist;
    }

    public String getAlbum() {
        return album;
    }

    public void setAlbum(String album) {
        this.album = album;
    }

    public String getYear() {
        return year;
    }

    public void setYear(String year) {
        year = year;
    }

    public LinkedList<String> getFeaturings() {
        return featurings;
    }

    public void setFeaturings(LinkedList<String> featurings) {
        this.featurings = featurings;
    }

    public String getGenre() {
        return genre;
    }

    public void setGenre(String genre) {
        this.genre = genre;
    }

    public Integer getTrackNumber() {
        return trackNumber;
    }

    public void setTrackNumber(Integer trackNumber) {
        this.trackNumber = trackNumber;
    }

    public Integer getTrackTotals() {
        return trackTotals;
    }

    public void setTrackTotals(Integer trackTotals) {
        this.trackTotals = trackTotals;
    }

    public Integer getDuration() {
        return duration;
    }

    public void setDuration(Integer duration) {
        this.duration = duration;
    }

    public TrackType getTrackType() {
        return trackType;
    }

    public void setTrackType(TrackType trackType) {
        this.trackType = trackType;
    }

    public String getImage() {
        return image;
    }

    public void setImage(String image) {
        this.image = image;
    }

    public String getUrl() {
        return url;
    }

    public void setUrl(String url) {
        this.url = url;
    }

    public String getFilename() {
        return filename;
    }

    public void setFilename(String filename) {
        this.filename = filename;
    }

    public String getFilenameTemp() {
        return filenameTemp;
    }

    public void setFilenameTemp(String filenameTemp) {
        this.filenameTemp = filenameTemp;
    }

    public String getFilenameExt() {
        return filenameExt;
    }

    public void setFilenameExt(String filenameExt) {
        this.filenameExt = filenameExt;
    }

    public String getSearchPattern() {
        return searchPattern;
    }

    public void setSearchPattern(String searchPattern) {
        this.searchPattern = searchPattern;
    }

    public String getLyrics() {
        return lyrics;
    }

    public void setLyrics(String lyrics) {
        this.lyrics = lyrics;
    }

    public Boolean getLocal() {
        return local;
    }

    public void setLocal(Boolean local) {
        local = local;
    }
}
