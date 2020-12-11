CREATE TABLE ks2.t1 (
    c1 BIGINT NOT NULL,
    c2 BIGINT NOT NULL,
    PRIMARY KEY (c1)
) ENGINE=Innodb;


CREATE TABLE ks1.c2_lookup (
    c2 BIGINT NOT NULL,
    keyspace_id binary(8),
    UNIQUE KEY (c1)
) ENGINE=Innodb;

