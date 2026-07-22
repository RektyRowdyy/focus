-- Volume is a 0..1 float in the API (and maps 1:1 to Web Audio gain). real (float32)
-- can't represent e.g. 0.6 exactly, so a PUT value != the GET value. Widen to double
-- precision so values round-trip cleanly against the float64 API / JS number.
alter table soundscape_prefs alter column volume type double precision;
