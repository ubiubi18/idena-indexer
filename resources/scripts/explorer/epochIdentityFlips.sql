select f.Cid,
       a.address              author,
       COALESCE(f.Status, '') status,
       COALESCE(f.Answer, '') answer,
       COALESCE(short.answers, 0),
       COALESCE(long.answers, 0),
       b.timestamp
from flips f
         join transactions t on t.id = f.tx_id
         join addresses a on a.id = t.from
         join blocks b on b.id = t.block_id
         join epochs e on e.id = b.epoch_id
         left join (select a.flip_id, count(*) answers from answers a where a.is_short = true group by a.flip_id) short
                   on short.flip_id = f.id
         left join (select a.flip_id, count(*) answers from answers a where a.is_short = false group by a.flip_id) long
                   on long.flip_id = f.id
where e.epoch = $1
  and lower(a.address) = lower($2)
order by b.height