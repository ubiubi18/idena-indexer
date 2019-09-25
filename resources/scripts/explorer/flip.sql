select a.address               author,
       f.size,
       b.timestamp,
       coalesce(f.answer, '')  answer,
       coalesce(f.status, '')  status,
       t.hash                  tx_hash,
       b.hash                  block_hash,
       b.height                block_height,
       b.epoch                 epoch,
       coalesce(fw.word_1, 0) word_id_1,
       coalesce(wd1.name, '')  word_name_1,
       coalesce(wd1.description, '')  word_desc_1,
       coalesce(fw.word_2, 0) word_id_2,
       coalesce(wd2.name, '')  word_name_2,
       coalesce(wd2.description, '')  word_desc_2
from flips f
         join transactions t on t.id = f.tx_id
         join blocks b on b.height = t.block_height
         join addresses a on a.id = t.from
         left join flip_words fw on fw.flip_id = f.id
         left join words_dictionary wd1 on wd1.id = fw.word_1
         left join words_dictionary wd2 on wd2.id = fw.word_2
where LOWER(f.cid) = LOWER($1)