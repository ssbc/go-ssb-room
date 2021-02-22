-- +migrate Up
CREATE TABLE notices (
  id   integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  title text NOT NULL,
  content text NOT NULL,
  language text NOT NULL
);

CREATE TABLE pinned_notices (
  name text NOT NULL,
  notice_id   integer NOT NULL UNIQUE,
  language text NOT NULL,

  PRIMARY KEY (name, language),
  
  -- make sure the notices exist
  FOREIGN KEY ( notice_id ) REFERENCES notices( "id" )
);

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

INSERT INTO pinned_notices (name, notice_id, language) VALUES
('NoticeDescription', 1, 'en-GB'),
('NoticeNews', 2, 'en-GB'),
('NoticeCodeOfConduct', 3, 'en-GB'),
('NoticePrivacyPolicy', 4, 'en-GB'),
('NoticePrivacyPolicy', 5, 'de-DE'),
('NoticeDescription', 6, 'de-DE');

-- +migrate Down
DROP TABLE notices;
DROP TABLE pinned_notices;