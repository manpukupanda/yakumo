CREATE TABLE documents (
  date char(10) NOT NULL,
  seqNumber int NOT NULL,
  docID char(8) NOT NULL,
  submitDateTime char(16) NULL,
  edinetCode char(6) NULL,
  secCode char(5) NULL,
  filerName text NULL,
  periodStart  char(10) NULL,
  periodEnd char(10) NULL,
  docDescription text NULL,
  PRIMARY KEY (date, seqNumber)
);


CREATE TABLE document_texts (
  docID char(8) NOT NULL,
  seq int NOT NULL,
  title text NOT NULL,
  breadcrumb text NOT NULL,
  content text NOT NULL,
  PRIMARY KEY (docID, seq)
);

CREATE EXTENSION IF NOT EXISTS pgroonga;
CREATE INDEX pgroonga_content_index ON document_texts USING pgroonga (breadcrumb, content);
