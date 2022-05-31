create table customers_sharded (
  id bigint,
  name varchar(64),
  age SMALLINT,
  primary key (id)
) Engine=InnoDB;

