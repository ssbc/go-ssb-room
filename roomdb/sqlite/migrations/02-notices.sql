-- SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
--
-- SPDX-License-Identifier: CC0-1.0

-- +migrate Up
CREATE TABLE pins (
  id integer NOT NULL PRIMARY KEY,
  name text NOT NULL UNIQUE
);


CREATE TABLE notices (
  id   integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  title text NOT NULL,
  content text NOT NULL,
  language text NOT NULL
);

-- n:m relation table
CREATE TABLE pin_notices (
  notice_id integer NOT NULL,
  pin_id integer NOT NULL,
  
  PRIMARY KEY (notice_id, pin_id),

  FOREIGN KEY ( notice_id ) REFERENCES notices( "id" ),
  FOREIGN KEY ( pin_id ) REFERENCES pins( "id" )
);

-- TODO: find a better way to insert the defaults
INSERT INTO pins (name) VALUES
('NoticeDescription'),
('NoticeNews'),
('NoticeCodeOfConduct'),
('NoticePrivacyPolicy');

INSERT INTO notices (title, content, language)  VALUES
('Description', 'Basic description of this Room.', 'en-GB'),
('News', 'Some recent updates...', 'en-GB'),
('Code of conduct', 'We expect each other to ...
* be considerate
* be respectful
* be responsible
* be dedicated
* be empathetic
', 'en-GB'),
('Privacy Policy', 'To be updated', 'en-GB'),
('Datenschutzrichtlinien', 'Bitte aktualisieren', 'de-DE'),
('Beschreibung', 'Allgemeine beschreibung des Raumes.', 'de-DE');

INSERT INTO pin_notices (notice_id, pin_id) VALUES
(1, 1),
(2, 2),
(3, 3),
(4, 4),
(5, 4),
(6, 1);

-- +migrate Down
DROP TABLE notices;
DROP TABLE pins;
DROP TABLE pin_notices;