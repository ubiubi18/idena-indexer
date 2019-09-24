select a.address
from bad_authors ba
         join epoch_identities ei on ei.id = ba.epoch_identity_id
         join address_states s on s.id = ei.address_state_id
         join addresses a on a.id = s.address_id
where ei.epoch = $1
limit $3
offset
$2