insert into block_validators (block_height, address_id)
values ($1, (select id from addresses where address = $2))